//go:build windows

package input

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows' notifications for the ping (plan 64, designs D4 and D5). A
// desktop program raises them under an app ID that a shortcut in the Start
// menu carries ("Without a valid shortcut installed in the Start screen or
// in All Programs, you cannot raise a toast notification from a desktop
// app": Microsoft Learn, "How to enable desktop toast notifications through
// an AppUserModelID"). The shortcut is made with IShellLinkW and its
// property store; the notifications go through the Windows Runtime's
// ToastNotificationManager (Windows.UI.Notifications), reached through
// combase.dll and method tables as the folder dialog reaches COM (plan 52).
// Plan 64's probe ran each call on this machine first.

var (
	combase                    = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize           = combase.NewProc("RoInitialize")
	procRoUninitialize         = combase.NewProc("RoUninitialize")
	procRoGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	procRoActivateInstance     = combase.NewProc("RoActivateInstance")
	procWindowsCreateString    = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString    = combase.NewProc("WindowsDeleteString")
	procPropVariantClear       = ole32.NewProc("PropVariantClear")
)

func guid(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

// The classes and interfaces called (ShObjIdl_core.h, propkey.h,
// windows.ui.notifications.h, windows.data.xml.dom.h).
var (
	clsidShellLink          = guid("{00021401-0000-0000-C000-000000000046}")
	iidShellLinkW           = guid("{000214F9-0000-0000-C000-000000000046}")
	iidPropertyStore        = guid("{886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99}")
	iidPersistFile          = guid("{0000010B-0000-0000-C000-000000000046}")
	iidToastManagerStatics  = guid("{50AC103F-D235-4598-BBEF-98FE4D1A3AD4}")
	iidToastManagerStatics2 = guid("{7AB93C52-0E48-4750-BA9D-1A4113981847}")
	iidToastFactory         = guid("{04124B20-82C6-4229-B109-FD9ED4662B53}")
	iidToast2               = guid("{9DFB9FD1-143A-490E-90BF-B9FBA7132DE7}")
	iidXMLDocument          = guid("{F7F3A506-1E87-42D6-BCFB-B8C809FA5494}")
	iidXMLDocumentIO        = guid("{6CD0E74E-EE65-4489-9EBF-CA43E87BA637}")
	pkeyAppUserModelID      = propertyKey{guid("{9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3}"), 5}
)

// The methods called, by their place in the method tables: IUnknown's
// QueryInterface, IShellLinkW's, IPropertyStore's, IPersistFile's, and the
// Windows Runtime interfaces', whose first six are IInspectable's.
const (
	mQueryInterface            = 0
	mLinkGetPath               = 3
	mLinkSetDescription        = 7
	mLinkGetArguments          = 10
	mLinkSetArguments          = 11
	mLinkSetPath               = 20
	mStoreGetValue             = 5
	mStoreSetValue             = 6
	mStoreCommit               = 7
	mFileLoad                  = 5
	mFileSave                  = 6
	mCreateToastNotifierWithID = 7 // IToastNotificationManagerStatics
	mGetHistory                = 6 // IToastNotificationManagerStatics2
	mCreateToastNotification   = 6 // IToastNotificationFactory
	mLoadXML                   = 6 // IXmlDocumentIO
	mPutTag                    = 6 // IToastNotification2
	mPutGroup                  = 8
	mNotifierShow              = 6 // IToastNotifier
	mRemoveGroupedTagWithID    = 8 // IToastNotificationHistory
	roInitMultithreaded        = 1
	hrChangedMode              = 0x80010106 // RPC_E_CHANGED_MODE: COM already in another apartment
	hrNotFound                 = 0x80070490 // HRESULT_FROM_WIN32(ERROR_NOT_FOUND)
	vtLPWSTR                   = 31
)

// terminalProfile is the profile the shortcut opens, scripts/acc.ps1's.
const terminalProfile = "Agent Command Center"

type propertyKey struct {
	fmtid windows.GUID
	pid   uint32
}

// propVariant is a PROPVARIANT holding a string, VT_LPWSTR.
type propVariant struct {
	vt  uint16
	_   [3]uint16
	str *uint16
	_   uintptr
}

func hrError(step string, hr uintptr) error {
	return fmt.Errorf("%s: HRESULT 0x%08X", step, uint32(hr))
}

// withRuntime runs fn on this goroutine's thread, locked for it, with the
// Windows Runtime and COM initialised for the call.
func withRuntime(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	switch hr, _, _ := procRoInitialize.Call(roInitMultithreaded); hr {
	case 0, 1: // S_OK, S_FALSE already initialised: undone as well
		defer procRoUninitialize.Call()
	case hrChangedMode:
	default:
		return hrError("RoInitialize", hr)
	}
	return fn()
}

// hstring is a Windows Runtime string of s; deleteHString frees it.
func hstring(s string) (uintptr, error) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return 0, err
	}
	var h uintptr
	if hr, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&h))); hr != 0 {
		return 0, hrError("WindowsCreateString", hr)
	}
	return h, nil
}

func deleteHString(h uintptr) { procWindowsDeleteString.Call(h) }

// hstrings makes one string of each of ss, all freed by the function it
// gives, or none and the first failure.
func hstrings(ss ...string) ([]uintptr, func(), error) {
	hs := make([]uintptr, 0, len(ss))
	free := func() {
		for _, h := range hs {
			deleteHString(h)
		}
	}
	for _, s := range ss {
		h, err := hstring(s)
		if err != nil {
			free()
			return nil, func() {}, err
		}
		hs = append(hs, h)
	}
	return hs, free, nil
}

func queryInterface(o *comObject, iid *windows.GUID, name string) (*comObject, error) {
	var p *comObject
	if hr := o.call(mQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&p))); hr != 0 {
		return nil, hrError("QueryInterface("+name+")", hr)
	}
	return p, nil
}

func activationFactory(class string, iid *windows.GUID) (*comObject, error) {
	h, err := hstring(class)
	if err != nil {
		return nil, err
	}
	defer deleteHString(h)
	var f *comObject
	if hr, _, _ := procRoGetActivationFactory.Call(h, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&f))); hr != 0 {
		return nil, hrError("RoGetActivationFactory("+class+")", hr)
	}
	return f, nil
}

// Notifier shows ACC's notifications under its app ID (design D4): one at a
// time, its tag and group always the same, so that a newer one replaces the
// older and Clear takes it out of Windows' list. Each call runs on the
// calling goroutine's thread; the dashboard makes them one at a time.
type Notifier struct{ appID string }

// Show puts up a notification, its title over its text, silent.
func (n *Notifier) Show(title, text string) error {
	return withRuntime(func() error {
		statics, err := activationFactory("Windows.UI.Notifications.ToastNotificationManager", &iidToastManagerStatics)
		if err != nil {
			return err
		}
		defer statics.release()
		hs, free, err := hstrings(n.appID, notifyTag, notifyGroup)
		if err != nil {
			return err
		}
		defer free()
		var notifier *comObject
		if hr := statics.call(mCreateToastNotifierWithID, hs[0], uintptr(unsafe.Pointer(&notifier))); hr != 0 {
			return hrError("CreateToastNotifierWithId", hr)
		}
		defer notifier.release()
		doc, err := loadXML(toastXML(title, text))
		if err != nil {
			return err
		}
		defer doc.release()
		factory, err := activationFactory("Windows.UI.Notifications.ToastNotification", &iidToastFactory)
		if err != nil {
			return err
		}
		defer factory.release()
		var toast *comObject
		if hr := factory.call(mCreateToastNotification, uintptr(unsafe.Pointer(doc)), uintptr(unsafe.Pointer(&toast))); hr != 0 {
			return hrError("CreateToastNotification", hr)
		}
		defer toast.release()
		toast2, err := queryInterface(toast, &iidToast2, "IToastNotification2")
		if err != nil {
			return err
		}
		defer toast2.release()
		if hr := toast2.call(mPutTag, hs[1]); hr != 0 {
			return hrError("put_Tag", hr)
		}
		if hr := toast2.call(mPutGroup, hs[2]); hr != 0 {
			return hrError("put_Group", hr)
		}
		if hr := notifier.call(mNotifierShow, uintptr(unsafe.Pointer(toast))); hr != 0 {
			return hrError("Show", hr)
		}
		return nil
	})
}

// Clear takes ACC's notification out of Windows' list; none there is no
// failure.
func (n *Notifier) Clear() error {
	return withRuntime(func() error {
		statics, err := activationFactory("Windows.UI.Notifications.ToastNotificationManager", &iidToastManagerStatics2)
		if err != nil {
			return err
		}
		defer statics.release()
		var history *comObject
		if hr := statics.call(mGetHistory, uintptr(unsafe.Pointer(&history))); hr != 0 {
			return hrError("get_History", hr)
		}
		defer history.release()
		hs, free, err := hstrings(notifyTag, notifyGroup, n.appID)
		if err != nil {
			return err
		}
		defer free()
		if hr := history.call(mRemoveGroupedTagWithID, hs[0], hs[1], hs[2]); hr != 0 && hr != hrNotFound {
			return hrError("Remove", hr)
		}
		return nil
	})
}

// loadXML is an XmlDocument holding content.
func loadXML(content string) (*comObject, error) {
	class, err := hstring("Windows.Data.Xml.Dom.XmlDocument")
	if err != nil {
		return nil, err
	}
	defer deleteHString(class)
	var inspectable *comObject
	if hr, _, _ := procRoActivateInstance.Call(class, uintptr(unsafe.Pointer(&inspectable))); hr != 0 {
		return nil, hrError("RoActivateInstance(XmlDocument)", hr)
	}
	defer inspectable.release()
	xmlIO, err := queryInterface(inspectable, &iidXMLDocumentIO, "IXmlDocumentIO")
	if err != nil {
		return nil, err
	}
	defer xmlIO.release()
	h, err := hstring(content)
	if err != nil {
		return nil, err
	}
	defer deleteHString(h)
	if hr := xmlIO.call(mLoadXML, h); hr != 0 {
		return nil, hrError("LoadXml", hr)
	}
	return queryInterface(inspectable, &iidXMLDocument, "IXmlDocument")
}

// NotifyPaths is the user's Start menu programs folder and Windows
// Terminal's alias, from Windows' known folders, which Windows reads with
// the process's APPDATA and LOCALAPPDATA (plan 64's check found them under
// a test's own).
func NotifyPaths() (dir, terminal string, err error) {
	if dir, err = windows.KnownFolderPath(windows.FOLDERID_Programs, 0); err != nil {
		return "", "", fmt.Errorf("the Start menu's folder: %w", err)
	}
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		return "", "", fmt.Errorf("the local app data folder: %w", err)
	}
	return dir, filepath.Join(local, "Microsoft", "WindowsApps", "wt.exe"), nil
}

// Notifications makes Windows' notifications ready in the user's own Start
// menu (SetUpNotifications on NotifyPaths) and gives their calls, or none
// and why. Its first call in a process takes about a second, the shell's
// libraries loading: the dashboard makes it in a command, off its first
// frame's path.
func Notifications() (show func(title, text string) error, clear func() error, problem string) {
	dir, terminal, err := NotifyPaths()
	if err != nil {
		return nil, nil, "notifications off: " + err.Error()
	}
	n, why := SetUpNotifications(dir, terminal)
	if n == nil {
		return nil, nil, why
	}
	return n.Show, n.Clear, ""
}

// SetUpNotifications makes sure ACC's shortcut is in dir, the user's Start
// menu programs folder, making it when missing, pointed at terminal,
// Windows Terminal's alias, with ACC's profile, and gives the Notifier for
// its app ID (design D5). An existing one is never rewritten; one of that
// name that is not ACC's (another app ID) is left as it is: no Notifier,
// and why.
func SetUpNotifications(dir, terminal string) (*Notifier, string) {
	path := filepath.Join(dir, NotifyShortcut+".lnk")
	problem := ""
	err := withRuntime(func() error {
		_, err := os.Lstat(path)
		switch {
		case err == nil:
			_, _, id, err := readShortcut(path)
			if err != nil {
				return err
			}
			if id != NotifyAppID {
				problem = fmt.Sprintf("notifications off: the Start menu's %q shortcut is not ACC's (its app ID is %q); left as it is", NotifyShortcut, id)
			}
			return nil
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
		if _, err := os.Lstat(terminal); err != nil {
			problem = "notifications off: Windows Terminal's wt.exe was not found (" + err.Error() + ")"
			return nil
		}
		return makeShortcut(path, terminal, `-p "`+terminalProfile+`"`, "Agent Command Center: its Windows Terminal profile", NotifyAppID)
	})
	if err != nil {
		return nil, "notifications off: " + err.Error()
	}
	if problem != "" {
		return nil, problem
	}
	return &Notifier{appID: NotifyAppID}, ""
}

func newShellLink() (*comObject, error) {
	var link *comObject
	if hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, clsctxInprocSrv, uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&link))); hr != 0 {
		return nil, hrError("CoCreateInstance(ShellLink)", hr)
	}
	return link, nil
}

func wide(s string) uintptr {
	p, _ := windows.UTF16PtrFromString(s)
	return uintptr(unsafe.Pointer(p))
}

// makeShortcut writes a shortcut to target with args and the app ID appID.
func makeShortcut(path, target, args, desc, appID string) error {
	link, err := newShellLink()
	if err != nil {
		return err
	}
	defer link.release()
	if hr := link.call(mLinkSetPath, wide(target)); hr != 0 {
		return hrError("SetPath", hr)
	}
	if hr := link.call(mLinkSetArguments, wide(args)); hr != 0 {
		return hrError("SetArguments", hr)
	}
	if hr := link.call(mLinkSetDescription, wide(desc)); hr != 0 {
		return hrError("SetDescription", hr)
	}
	store, err := queryInterface(link, &iidPropertyStore, "IPropertyStore")
	if err != nil {
		return err
	}
	defer store.release()
	id, err := windows.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	pv := propVariant{vt: vtLPWSTR, str: id}
	if hr := store.call(mStoreSetValue, uintptr(unsafe.Pointer(&pkeyAppUserModelID)), uintptr(unsafe.Pointer(&pv))); hr != 0 {
		return hrError("SetValue(System.AppUserModel.ID)", hr)
	}
	runtime.KeepAlive(id)
	if hr := store.call(mStoreCommit); hr != 0 {
		return hrError("Commit", hr)
	}
	file, err := queryInterface(link, &iidPersistFile, "IPersistFile")
	if err != nil {
		return err
	}
	defer file.release()
	if hr := file.call(mFileSave, wide(path), 1); hr != 0 { // fRemember
		return hrError("Save", hr)
	}
	return nil
}

// readShortcut is a shortcut's target, arguments and app ID as Windows
// reads them.
func readShortcut(path string) (target, args, appID string, err error) {
	link, err := newShellLink()
	if err != nil {
		return "", "", "", err
	}
	defer link.release()
	file, err := queryInterface(link, &iidPersistFile, "IPersistFile")
	if err != nil {
		return "", "", "", err
	}
	defer file.release()
	if hr := file.call(mFileLoad, wide(path), 0); hr != 0 { // STGM_READ
		return "", "", "", hrError("Load", hr)
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	if hr := link.call(mLinkGetPath, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 0); hr != 0 {
		return "", "", "", hrError("GetPath", hr)
	}
	target = windows.UTF16ToString(buf)
	if hr := link.call(mLinkGetArguments, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); hr != 0 {
		return "", "", "", hrError("GetArguments", hr)
	}
	args = windows.UTF16ToString(buf)
	store, err := queryInterface(link, &iidPropertyStore, "IPropertyStore")
	if err != nil {
		return "", "", "", err
	}
	defer store.release()
	var pv propVariant
	if hr := store.call(mStoreGetValue, uintptr(unsafe.Pointer(&pkeyAppUserModelID)), uintptr(unsafe.Pointer(&pv))); hr != 0 {
		return "", "", "", hrError("GetValue(System.AppUserModel.ID)", hr)
	}
	if pv.vt == vtLPWSTR && pv.str != nil {
		appID = windows.UTF16PtrToString(pv.str)
	}
	procPropVariantClear.Call(uintptr(unsafe.Pointer(&pv)))
	return target, args, appID, nil
}

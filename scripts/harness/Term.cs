using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading;
using System.Windows.Automation;
using System.Windows.Automation.Text;

// Term drives a real terminal window: focus, keyboard and mouse through
// SendInput, the screen through UI Automation. Every input batch first checks
// the target window has the focus, and stops otherwise.
public static class Term {
    [DllImport("user32.dll")] static extern bool SetProcessDpiAwarenessContext(IntPtr v);
    [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] static extern bool SetForegroundWindow(IntPtr h);
    [DllImport("user32.dll")] static extern bool ShowWindow(IntPtr h, int n);
    [DllImport("user32.dll")] static extern bool BringWindowToTop(IntPtr h);
    [DllImport("user32.dll")] static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] static extern int GetSystemMetrics(int i);
    // Glide sends real pointer moves (SendInput, absolute): SetCursorPos alone
    // gives Windows Terminal no pointer-move events, so its own drag never starts.
    public static void Glide(int x, int y) {
        int vx = GetSystemMetrics(76), vy = GetSystemMetrics(77), vw = GetSystemMetrics(78), vh = GetSystemMetrics(79);
        var i = new INPUT { type = 0 };
        i.u.mi.dx = (int)(((x - vx) * 65535L + vw / 2) / (vw - 1)); i.u.mi.dy = (int)(((y - vy) * 65535L + vh / 2) / (vh - 1));
        i.u.mi.flags = 0x0001 | 0x8000 | 0x4000;
        Send(i);
    }
    [DllImport("user32.dll")] static extern uint SendInput(uint n, INPUT[] i, int size);
    [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc f, IntPtr l);
    [DllImport("user32.dll")] static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll")] static extern int GetClassName(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h, out RECT r);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
    delegate bool EnumProc(IntPtr h, IntPtr l);

    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L, T, R, B; }
    [StructLayout(LayoutKind.Sequential)] struct MOUSEINPUT { public int dx, dy; public uint data, flags, time; public IntPtr extra; }
    [StructLayout(LayoutKind.Sequential)] struct KEYBDINPUT { public ushort vk, scan; public uint flags, time; public IntPtr extra; }
    [StructLayout(LayoutKind.Explicit)] struct UNION { [FieldOffset(0)] public MOUSEINPUT mi; [FieldOffset(0)] public KEYBDINPUT ki; }
    [StructLayout(LayoutKind.Sequential)] struct INPUT { public uint type; public UNION u; }

    const uint KEYUP = 2, UNICODE = 4, EXTENDED = 1;
    const uint M_LEFTDOWN = 2, M_LEFTUP = 4, M_WHEEL = 0x800;

    public static IntPtr Target = IntPtr.Zero;
    public static int KeyDelay = 15;
    public static bool Scan = false;
    [DllImport("user32.dll")] static extern uint MapVirtualKey(uint c, uint t);
    public static int Refocus = 0;

    public static void Dpi() { SetProcessDpiAwarenessContext(new IntPtr(-4)); }

    public class Win { public IntPtr H; public string Title, Class; public uint Pid; public override string ToString() { return string.Format("{0:X} [{1}] {2} pid {3}", H.ToInt64(), Class, Title, Pid); } }

    public static List<Win> Windows() {
        var list = new List<Win>();
        EnumWindows((h, l) => {
            if (!IsWindowVisible(h)) return true;
            var t = new StringBuilder(512); GetWindowText(h, t, 512);
            var c = new StringBuilder(256); GetClassName(h, c, 256);
            uint pid; GetWindowThreadProcessId(h, out pid);
            list.Add(new Win { H = h, Title = t.ToString(), Class = c.ToString(), Pid = pid });
            return true;
        }, IntPtr.Zero);
        return list;
    }

    public static RECT Rect(IntPtr h) { RECT r; GetWindowRect(h, out r); return r; }

    // Focus brings the target forward; Alt released first lifts the
    // foreground lock Windows puts on background processes.
    public static bool Focus() {
        for (int i = 0; i < 20; i++) {
            if (GetForegroundWindow() == Target) return true;
            ShowWindow(Target, 9);
            Send(Key(0x12, false), Key(0x12, true));
            BringWindowToTop(Target);
            SetForegroundWindow(Target);
            Thread.Sleep(100);
        }
        return GetForegroundWindow() == Target;
    }

    static void Guard() {
        if (Target == IntPtr.Zero) throw new Exception("no target window");
        if (GetForegroundWindow() != Target) Refocus++;
        if (GetForegroundWindow() != Target && !Focus())
            throw new Exception("the target window lost the focus: input stopped");
    }

    static INPUT Key(ushort vk, bool up) {
        var i = new INPUT { type = 1 };
        i.u.ki.vk = vk;
        i.u.ki.flags = up ? KEYUP : 0;
        if (Scan) i.u.ki.scan = (ushort)MapVirtualKey(vk, 0);
        if (vk == 0x21 || vk == 0x22 || vk == 0x23 || vk == 0x24 || (vk >= 0x25 && vk <= 0x28) || vk == 0x2D || vk == 0x2E) i.u.ki.flags |= EXTENDED;
        return i;
    }
    static INPUT Uni(char c, bool up) {
        var i = new INPUT { type = 1 };
        i.u.ki.scan = c;
        i.u.ki.flags = UNICODE | (up ? KEYUP : 0);
        return i;
    }
    static void Send(params INPUT[] ins) {
        if (SendInput((uint)ins.Length, ins, Marshal.SizeOf(typeof(INPUT))) != ins.Length)
            throw new Exception("SendInput was blocked");
    }

    // Chord presses modifiers (VK codes) then the key, and releases them.
    public static void Chord(ushort[] mods, ushort vk) {
        Guard();
        var ins = new List<INPUT>();
        foreach (var m in mods) ins.Add(Key(m, false));
        ins.Add(Key(vk, false)); ins.Add(Key(vk, true));
        for (int i = mods.Length - 1; i >= 0; i--) ins.Add(Key(mods[i], true));
        Send(ins.ToArray());
        Thread.Sleep(KeyDelay * 4);
    }
    public static void Press(ushort vk) { Chord(new ushort[0], vk); }

    // Type sends text as Unicode key events, one character at a time; a tab
    // is the Tab key. Newlines are not typed: they are keys of their own.
    public static void Type(string s) {
        Guard();
        foreach (char c in s) {
            if (c == '\n' || c == '\r') throw new Exception("Type: newline; use a key");
            if (c == '\t') { Send(Key(0x09, false), Key(0x09, true)); }
            else { Send(Uni(c, false), Uni(c, true)); }
            Thread.Sleep(KeyDelay);
        }
    }

    public static void MoveTo(int x, int y) { Guard(); SetCursorPos(x, y); Thread.Sleep(40); }
    static INPUT Mouse(uint flags, uint data) { var i = new INPUT { type = 0 }; i.u.mi.flags = flags; i.u.mi.data = data; return i; }
    public static void LeftDown() { Guard(); Send(Mouse(M_LEFTDOWN, 0)); Thread.Sleep(40); }
    public static void RightClick(int x, int y) { MoveTo(x, y); Guard(); Send(Mouse(8, 0)); Thread.Sleep(40); Send(Mouse(16, 0)); Thread.Sleep(40); }
    public static void LeftUp() { Guard(); Send(Mouse(M_LEFTUP, 0)); Thread.Sleep(40); }
    public static void Click(int x, int y) { MoveTo(x, y); LeftDown(); LeftUp(); }
    // CtrlClick is a left click with Ctrl held, its key down before the
    // button and up after it, released whatever happens (plan 50). CtrlHold
    // sends Ctrl's key-down n times, as a held key repeats, then its key-up.
    public static void CtrlClick(int x, int y) {
        MoveTo(x, y);
        Guard(); Send(Key(0x11, false)); Thread.Sleep(40);
        try { LeftDown(); LeftUp(); } finally { Send(Key(0x11, true)); Thread.Sleep(40); }
    }
    public static void CtrlHold(int n) {
        Guard();
        try { for (int i = 0; i < n; i++) { Send(Key(0x11, false)); Thread.Sleep(40); } }
        finally { Send(Key(0x11, true)); Thread.Sleep(40); }
    }
    public static void Wheel(int x, int y, int notches) { MoveTo(x, y); Send(Mouse(M_WHEEL, (uint)(notches * 120))); Thread.Sleep(80); }
    public static void Drag(int x0, int y0, int x1, int y1) {
        MoveTo(x0, y0); LeftDown();
        for (int k = 1; k <= 8; k++) { Glide(x0 + (x1 - x0) * k / 8, y0 + (y1 - y0) * k / 8); Thread.Sleep(30); }
        Thread.Sleep(60); LeftUp();
    }

    // The terminal's text, through UI Automation's TextPattern.
    static AutomationElement TextHost(IntPtr h) {
        var root = AutomationElement.FromHandle(h);
        if ((bool)root.GetCurrentPropertyValue(AutomationElement.IsTextPatternAvailableProperty)) return root;
        var cond = new PropertyCondition(AutomationElement.IsTextPatternAvailableProperty, true);
        AutomationElement first = null;
        foreach (AutomationElement e in root.FindAll(TreeScope.Descendants, cond)) {
            if (e.Current.ClassName == "TextBlock") continue; // WT's tab title
            if (first == null) first = e;
        }
        return first;
    }
    public static string ScreenText(IntPtr h) {
        var e = TextHost(h);
        if (e == null) return null;
        var tp = (TextPattern)e.GetCurrentPattern(TextPattern.Pattern);
        return tp.DocumentRange.GetText(-1);
    }
    // Find returns the screen rectangles of the first occurrence of s.
    public static System.Windows.Rect[] Find(IntPtr h, string s) {
        var e = TextHost(h);
        if (e == null) return null;
        var tp = (TextPattern)e.GetCurrentPattern(TextPattern.Pattern);
        var r = tp.DocumentRange.FindText(s, false, false);
        if (r == null) return new System.Windows.Rect[0];
        return r.GetBoundingRectangles();
    }
    // Cells returns the rectangle of the visible text area and its first line.
    public static System.Windows.Rect[] Visible(IntPtr h) {
        var e = TextHost(h);
        var tp = (TextPattern)e.GetCurrentPattern(TextPattern.Pattern);
        var vr = tp.GetVisibleRanges();
        var list = new List<System.Windows.Rect>();
        foreach (var r in vr) list.AddRange(r.GetBoundingRectangles());
        return list.ToArray();
    }

    [DllImport("user32.dll")] static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint f);
    public static void Place(IntPtr h, int x, int y, int w, int ht) { ShowWindow(h, 9); Thread.Sleep(200); SetWindowPos(h, IntPtr.Zero, x, y, w, ht, 0x0004); }
    public static void Show(IntPtr h, int cmd) { ShowWindow(h, cmd); }
    [DllImport("user32.dll")] static extern bool IsZoomed(IntPtr h);
    public static bool Zoomed(IntPtr h) { return IsZoomed(h); }
    public static void Close(IntPtr h) { PostMessage(h, 0x0010, IntPtr.Zero, IntPtr.Zero); }
    public static bool Foreground(IntPtr h) { return GetForegroundWindow() == h; }

    [DllImport("user32.dll")] static extern bool IsHungAppWindow(IntPtr h);
    [DllImport("user32.dll")] static extern IntPtr SendMessageTimeout(IntPtr h, uint m, IntPtr w, IntPtr l, uint f, uint ms, out IntPtr r);
    [DllImport("user32.dll")] static extern short GetAsyncKeyState(int vk);
    // Mods lists the modifier keys Windows reports as down.
    public static string Mods() { var s = ""; foreach (var vk in new int[] { 0x10, 0x11, 0x12, 0x5B, 0x5C, 0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0x14, 0x90, 0x91 }) if ((GetAsyncKeyState(vk) & 0x8000) != 0) s += string.Format("{0:X} ", vk); return s == "" ? "none" : s; }
    // Responds says whether the window thread answers a message within ms.
    public static string Responds(IntPtr h, uint ms) { IntPtr r; var ok = SendMessageTimeout(h, 0, IntPtr.Zero, IntPtr.Zero, 2, ms, out r) != IntPtr.Zero; return "hung " + IsHungAppWindow(h) + ", answers WM_NULL " + ok; }
    public static void KeyDown(ushort vk) { Guard(); Send(Key(vk, false)); Thread.Sleep(30); }
    public static void KeyUp(ushort vk) { Send(Key(vk, true)); Thread.Sleep(30); }
    // Clicks presses n times on one point, quickly enough for a multi-click.
    public static void Clicks(int x, int y, int n) {
        MoveTo(x, y);
        for (int i = 0; i < n; i++) { Send(Mouse(M_LEFTDOWN, 0)); Thread.Sleep(30); Send(Mouse(M_LEFTUP, 0)); Thread.Sleep(60); }
    }

    // VisibleText is the text UI Automation reports as on screen.
    public static string VisibleText(IntPtr h) {
        var e = TextHost(h);
        if (e == null) return null;
        var tp = (TextPattern)e.GetCurrentPattern(TextPattern.Pattern);
        var sb = new StringBuilder();
        foreach (var r in tp.GetVisibleRanges()) sb.Append(r.GetText(-1));
        return sb.ToString();
    }
    // InvokeButton presses the first button of the window whose name
    // contains s; false when there is none.
    public static string InvokeButton(IntPtr h, string s) {
        var root = AutomationElement.FromHandle(h);
        var cond = new PropertyCondition(AutomationElement.ControlTypeProperty, ControlType.Button);
        foreach (AutomationElement e in root.FindAll(TreeScope.Descendants, cond)) {
            var n = e.Current.Name ?? "";
            if (n.IndexOf(s, StringComparison.OrdinalIgnoreCase) < 0) continue;
            object p;
            if (e.TryGetCurrentPattern(InvokePattern.Pattern, out p)) { ((InvokePattern)p).Invoke(); return n; }
        }
        return null;
    }
    public static string Buttons(IntPtr h) {
        var root = AutomationElement.FromHandle(h);
        var cond = new PropertyCondition(AutomationElement.ControlTypeProperty, ControlType.Button);
        var sb = new StringBuilder();
        foreach (AutomationElement e in root.FindAll(TreeScope.Descendants, cond)) sb.Append("[" + e.Current.Name + "]");
        return sb.ToString();
    }
}

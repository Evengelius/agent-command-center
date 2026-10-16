package proc

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The helpers are this test binary started again with ACC_PROC set.
func TestMain(m *testing.M) {
	switch os.Getenv("ACC_PROC") {
	case "":
		os.Exit(m.Run())
	case "dashboard": // starts a CLI in its job object, then waits
		p, err := Exec{}.Spawn(Spec{Path: os.Args[0], Env: append(os.Environ(), "ACC_PROC=cli")})
		if err != nil {
			fmt.Println("error", err)
			os.Exit(2)
		}
		fmt.Println("cli", p.Pid())
		time.Sleep(60 * time.Second)
	case "cli": // ignores everything
		time.Sleep(60 * time.Second)
	}
	os.Exit(0)
}

// TestJobKillsTreeWithDashboard is proposal step 2, item 3: when the
// dashboard's process is killed, without any cleanup of its own, the CLI it
// started is gone within 1 s (ASM-19).
func TestJobKillsTreeWithDashboard(t *testing.T) {
	dash := exec.Command(os.Args[0])
	dash.Env = append(os.Environ(), "ACC_PROC=dashboard")
	out, err := dash.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := dash.Start(); err != nil {
		t.Fatal(err)
	}
	defer dash.Wait()
	line, err := bufio.NewReader(out).ReadString('\n')
	f := strings.Fields(line)
	if err != nil || len(f) != 2 || f[0] != "cli" {
		dash.Process.Kill()
		t.Fatalf("dashboard said %q (%v)", line, err)
	}
	pid, _ := strconv.Atoi(f[1])
	if !alive(pid) {
		t.Fatal("the CLI is not running")
	}
	t0 := time.Now()
	if err := dash.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	for alive(pid) {
		if time.Since(t0) > time.Second {
			t.Fatalf("CLI %d still running 1 s after the dashboard was killed", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("CLI gone %v after the kill", time.Since(t0).Round(time.Millisecond))
}

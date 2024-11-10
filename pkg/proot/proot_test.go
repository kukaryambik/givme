package proot_test

import (
	"reflect"
	"testing"

	"github.com/kukaryambik/givme/pkg/proot"
)

func TestCmdBasic(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath:    "proot",
		Command:    []string{"ls", "-la"},
		MixedMode:  true,
		KillOnExit: true,
		Verbose:    2,
		Workdir:    "/home/user",
		ExtraFlags: []string{"--some-extra-flag"},
	}

	cmd := cfg.Cmd()

	expectedArgs := []string{
		"proot",
		"--some-extra-flag",
		"--mixed-mode true",
		"--kill-on-exit",
		"--verbose=2",
		"--cwd=/home/user",
		"ls",
		"-la",
	}

	if !reflect.DeepEqual(cmd.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, cmd.Args)
	}
}

func TestCmdWithEnv(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath:               "proot",
		DontPolluteRootfs:     true,
		ForceForeignBinary:    true,
		LibraryPath:           "/custom/lib",
		IgnoreMissingBindings: true,
	}

	cmd := cfg.Cmd()

	expectedEnv := []string{
		"PROOT_DONT_POLLUTE_ROOTFS=1",
		"PROOT_FORCE_FOREIGN_BINARY=1",
		"LD_LIBRARY_PATH=/custom/lib",
		"PROOT_IGNORE_MISSING_BINDINGS=1",
	}

	// Convert cmd.Env slice to map for easier comparison
	envMap := make(map[string]struct{})
	for _, env := range cmd.Env {
		envMap[env] = struct{}{}
	}

	for _, expected := range expectedEnv {
		if _, exists := envMap[expected]; !exists {
			t.Errorf("Expected environment variable %s not found", expected)
		}
	}
}

func TestCmdBindsAndPorts(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath: "proot",
		Binds: []string{
			"/host/path1:/guest/path1",
			"/host/path2:/guest/path2",
		},
		Ports: []string{
			"8080:80",
			"8443:443",
		},
	}

	cmd := cfg.Cmd()

	expectedArgs := []string{
		"proot",
		"--mixed-mode false",
		"--bind=/host/path1:/guest/path1",
		"--bind=/host/path2:/guest/path2",
		"--port=8080:80",
		"--port=8443:443",
	}

	// Since the order of flags can vary, we'll check for their presence
	argsMap := make(map[string]struct{})
	for _, arg := range cmd.Args {
		argsMap[arg] = struct{}{}
	}

	for _, expected := range expectedArgs {
		if _, exists := argsMap[expected]; !exists {
			t.Errorf("Expected argument %s not found", expected)
		}
	}
}

func TestCmdChangeID(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath:  "proot",
		ChangeID: "1000:1000",
	}

	cmd := cfg.Cmd()

	expectedArg := "--change-id=1000:1000"

	if len(cmd.Args) < 3 || cmd.Args[2] != expectedArg {
		t.Errorf("Expected argument %s, got %v", expectedArg, cmd.Args)
	}
}

func TestCmdInvalidChangeID(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath:  "proot",
		ChangeID: "invalid:id",
	}

	cmd := cfg.Cmd()

	// Since the ChangeID is invalid, it should not be added to cmd.Args
	for _, arg := range cmd.Args {
		if arg == "--change-id=invalid:id" {
			t.Errorf("Invalid change-id should not be added to arguments")
		}
	}
}

func TestCmdWithNoSeccomp(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath:   "proot",
		NoSeccomp: true,
	}

	cmd := cfg.Cmd()

	expectedEnv := "PROOT_NO_SECCOMP=1"

	found := false
	for _, env := range cmd.Env {
		if env == expectedEnv {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected environment variable %s not found", expectedEnv)
	}
}

func TestCmdFullConfig(t *testing.T) {
	cfg := &proot.ProotConf{
		BinPath:       "proot",
		Command:       []string{"bash"},
		MixedMode:     true,
		KillOnExit:    true,
		Link2Symlink:  true,
		Netcoop:       true,
		Verbose:       3,
		Workdir:       "/tmp",
		RootFS:        "/new/rootfs",
		KernelRelease: "5.10.0",
		Binds: []string{
			"/mnt/host1:/mnt/guest1",
			"/mnt/host2:/mnt/guest2",
		},
		Ports: []string{
			"8000:80",
			"8443:443",
		},
		DontPolluteRootfs:     true,
		ForceForeignBinary:    true,
		ForceKompat:           true,
		IgnoreMissingBindings: true,
		LibraryPath:           "/usr/local/lib",
		Loader:                "/custom/loader",
		NoSeccomp:             true,
		TmpDir:                "/tmp/proot",
		ExtraFlags:            []string{"--additional-flag"},
	}

	cmd := cfg.Cmd()

	// Expected arguments
	expectedArgs := []string{
		"proot",
		"--additional-flag",
		"--mixed-mode true",
		"--kill-on-exit",
		"--link2symlink",
		"--netcoop",
		"--verbose=3",
		"--cwd=/tmp",
		"--rootfs=/new/rootfs",
		"--kernel-release=5.10.0",
		"--bind=/mnt/host1:/mnt/guest1",
		"--bind=/mnt/host2:/mnt/guest2",
		"--port=8000:80",
		"--port=8443:443",
		"bash",
	}

	// Expected environment variables
	expectedEnv := []string{
		"PROOT_DONT_POLLUTE_ROOTFS=1",
		"PROOT_FORCE_FOREIGN_BINARY=1",
		"PROOT_FORCE_KOMPAT=1",
		"PROOT_IGNORE_MISSING_BINDINGS=1",
		"LD_LIBRARY_PATH=/usr/local/lib",
		"PROOT_LOADER=/custom/loader",
		"PROOT_NO_SECCOMP=1",
		"PROOT_TMP_DIR=/tmp/proot",
	}

	// Check arguments
	argsMap := make(map[string]struct{})
	for _, arg := range cmd.Args {
		argsMap[arg] = struct{}{}
	}

	for _, expected := range expectedArgs {
		if _, exists := argsMap[expected]; !exists {
			t.Errorf("Expected argument %s not found", expected)
		}
	}

	// Check environment variables
	envMap := make(map[string]struct{})
	for _, env := range cmd.Env {
		envMap[env] = struct{}{}
	}

	for _, expected := range expectedEnv {
		if _, exists := envMap[expected]; !exists {
			t.Errorf("Expected environment variable %s not found", expected)
		}
	}
}

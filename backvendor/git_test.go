// Copyright (C) 2018 Tim Waugh
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package backvendor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/vcs"
)

const (
	envHelper     = "GO_WANT_HELPER_PROCESS"
	envStdout     = "STDOUT"
	envStderr     = "STDERR"
	envExitStatus = "EXIT_STATUS"
)

var mockedExitStatus int
var mockedStdout, mockedStderr string

// Capture exec.Command calls via execCommand and make them run our
// fake version instead. This returns a function which the caller
// should defer a call to in order to reset execCommand.
func mockExecCommand() func() {
	execCommand = fakeExecCommand

	// Reset it afterwards
	return func() {
		execCommand = exec.Command
		mockedExitStatus = 0
		mockedStdout = ""
		mockedStderr = ""
	}
}

// Run this test binary (again!) but transfer control immediately to
// TestHelper, telling it how to act.
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	testBinary := os.Args[0]
	opts := []string{"-test.run=TestHelper", "--", command}
	opts = append(opts, args...)
	cmd := exec.Command(testBinary, opts...)
	cmd.Env = []string{
		envHelper + "=1",
		envStdout + "=" + mockedStdout,
		envStderr + "=" + mockedStderr,
		envExitStatus + "=" + strconv.Itoa(mockedExitStatus),
	}
	return cmd
}

// This runs in its own process (see fakeExecCommand) and mocks the
// command being run.
func TestHelper(t *testing.T) {
	if os.Getenv(envHelper) != "1" {
		return
	}
	fmt.Print(os.Getenv(envStdout))
	fmt.Fprint(os.Stderr, os.Getenv(envStderr))
	exit, _ := strconv.Atoi(os.Getenv(envExitStatus))
	os.Exit(exit)
}

func TestGitRevisions(t *testing.T) {
	defer mockExecCommand()()

	wt := WorkingTree{
		Source: &GoSource{},
		VCS:    vcs.ByCmd("git"),
	}

	expectedRevs := []string{
		"d4c3dbfa77a74ae238e401d5d2197b45f30d8513",
		"a2176f4275f92ceddb47cff1e363313156124bf6",
	}
	mockedStdout = strings.Join(expectedRevs, "\n") + "\n"

	revs, err := wt.Revisions()
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != len(expectedRevs) {
		t.Fatalf("wrong number of revisions: got %d, want %d",
			len(revs), len(expectedRevs))
	}
	for i, rev := range expectedRevs {
		if revs[i] != rev {
			t.Fatalf("unexpected revisions: got %v, want %v",
				revs, expectedRevs)
		}
	}
}

func TestGitRevisionsError(t *testing.T) {
	defer mockExecCommand()()

	wt := WorkingTree{
		Source: &GoSource{},
		VCS:    vcs.ByCmd("git"),
	}

	mockedStderr = "fatal: not a git repository\n"
	mockedExitStatus = 128

	_, err := wt.Revisions()
	if err == nil {
		t.Fatal("git failure was not reported")
	}
}

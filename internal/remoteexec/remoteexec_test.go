package remoteexec

import "testing"

func TestQuote_EscapesShellMetacharacters(t *testing.T) {
	got := Quote([]string{"cat", "/tmp/it's; $(rm -rf /).txt"})
	want := "'cat' '/tmp/it'\\''s; $(rm -rf /).txt'"
	if got != want {
		t.Errorf("Quote = %q, want %q", got, want)
	}
}

func TestRun_LocalUsesArgvDirectly(t *testing.T) {
	cmd := Run("", "echo", "hi")
	if cmd.Args[0] != "echo" || len(cmd.Args) != 2 || cmd.Args[1] != "hi" {
		t.Errorf("Run local Args = %v, want [echo hi]", cmd.Args)
	}
}

func TestRun_RemoteQuotesIntoSingleArg(t *testing.T) {
	cmd := Run("host", "echo", "a b")
	if cmd.Path == "" || len(cmd.Args) != 3 || cmd.Args[1] != "host" {
		t.Fatalf("Run remote Args = %v, want [ssh host ...]", cmd.Args)
	}
	if cmd.Args[2] != "'echo' 'a b'" {
		t.Errorf("Run remote command = %q, want %q", cmd.Args[2], "'echo' 'a b'")
	}
}

package hookcd

import "testing"

func TestAbsoluteCDClaimChain(t *testing.T) {
	for _, command := range []string{
		"cd /abs/repo && achta claim take --repo /abs/repo --slice X",
		"cd /abs/repo\nachta claim take --repo /abs/repo --slice X",
	} {
		assertHookCommand(t, command, Allow)
	}
}

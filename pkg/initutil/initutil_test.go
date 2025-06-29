package initutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractWebValues(t *testing.T) {
	mockConf := `
TSD_WEBSER1=http://10.116.145.15/TsdWebService/TestUnitSequenceService.asmx=
TEXPERT_SERVER2=http://10.116.145.56/TexpertWebService/TsdTexpert.asmx=
XMPP_WEBSER1=10.116.105.85:5222=
TSD_WEBSER2=http://fake-address
`

	keys := []string{"TSD_WEBSER1", "XMPP_WEBSER1", "TEXPERT_SERVER2"}
	want := []string{
		"http://10.116.145.15/TsdWebService/TestUnitSequenceService.asmx",
		"10.116.105.85:5222",
		"http://10.116.145.56/TexpertWebService/TsdTexpert.asmx",
	}
	got := extractWebValues(mockConf, keys)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractIntConfig(t *testing.T) {
	conf := `
loop=300;
loopinterval=15;
some_other_key=999;
# loop=should_be_ignored
`

	tests := []struct {
		name       string
		key        string
		defaultVal int
		want       int
	}{
		{"loop present", "loop", 480, 300},
		{"interval present", "loopinterval", 60, 15},
		{"missing key fallback", "nonexistent", 42, 42},
		{"invalid value fallback", "some_other_key", 1000, 999}, // still valid int
		{"commented-out key", "#loop", 123, 123},                // shouldn't match commented line
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIntConfig(conf, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("key=%q expect=%d got=%d", tt.key, tt.want, got)
			}
		})
	}
}
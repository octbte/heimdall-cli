//go:build darwin

package tail

import (
	"strings"
	"testing"
)

func TestGenerateLaunchdPlist(t *testing.T) {
	plist := generateLaunchdPlist(
		"/usr/local/bin/heimdall",
		[]string{"/var/log/app.log"},
		"hm_live_key",
	)

	if !strings.Contains(plist, "com.heimdall.tail") {
		t.Errorf("missing label in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "/usr/local/bin/heimdall") {
		t.Errorf("missing exec path in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "HEIMDALL_INGEST_API_KEY") {
		t.Errorf("missing env key in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "hm_live_key") {
		t.Errorf("missing ingest key value in plist:\n%s", plist)
	}
	if !strings.Contains(plist, "<true/>") {
		t.Errorf("missing KeepAlive or RunAtLoad in plist:\n%s", plist)
	}
}

package rutte

import "testing"

func TestModURL(t *testing.T) {
	tests := []struct {
		urlIn       string
		expectedURL string

		pathCheckOutput bool
	}{
		{
			urlIn:           "./release-notes-0.5/#something",
			expectedURL:     "./release-notes-0.5.md#something",
			pathCheckOutput: true,
		},
		{
			urlIn:           "./release-notes-0.5/",
			expectedURL:     "./release-notes-0.5.md",
			pathCheckOutput: true,
		},
		{
			urlIn:           "../../../configuration/acme/http01/",
			expectedURL:     "../../configuration/acme/http01/README.md",
			pathCheckOutput: false,
		},
		{
			urlIn:           "../../configuration/acme/dns01/#setting-nameservers-for-dns01-self-check",
			expectedURL:     "../configuration/acme/dns01/README.md#setting-nameservers-for-dns01-self-check",
			pathCheckOutput: false,
		},
		{
			urlIn:           "../../configuration/acme/dns01/#setting-nameservers-for-dns01-self-check",
			expectedURL:     "../configuration/acme/dns01.md#setting-nameservers-for-dns01-self-check",
			pathCheckOutput: true,
		},
		{
			urlIn:           "../configuration/acme/dns01/#setting-nameservers-for-dns01-self-check",
			expectedURL:     "../configuration/acme/dns01.md#setting-nameservers-for-dns01-self-check",
			pathCheckOutput: true,
		},
		{
			urlIn:           "../",
			expectedURL:     "../README.md",
			pathCheckOutput: false,
		},
	}

	for _, test := range tests {
		urlOut, err := ModURL(test.urlIn, func(string) bool { return test.pathCheckOutput })
		if err != nil {
			t.Fatalf("didn't expect an error but got: %s", err)
			continue
		}

		if urlOut != test.expectedURL {
			t.Fatalf("wanted %q but got %q", test.expectedURL, urlOut)
			continue
		}
	}
}

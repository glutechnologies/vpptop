package command

import "testing"

func TestRemoteNodeAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "IPv4", input: "192.0.2.10", want: "192.0.2.10:7878"},
		{name: "IPv6", input: "2001:db8::10", want: "[2001:db8::10]:7878"},
		{name: "hostname rejected", input: "worker-1", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := remoteNodeAddress(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("remoteNodeAddress(%q) returned no error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("remoteNodeAddress(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("remoteNodeAddress(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

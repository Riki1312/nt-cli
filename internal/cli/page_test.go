package cli

import "testing"

func TestReplacePageContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		existing  string
		oldStr    string
		newStr    string
		want      string
		wantErr   bool
		firstOnly bool
	}{
		{
			name:     "replaces all exact matches",
			existing: "alpha beta alpha",
			oldStr:   "alpha",
			newStr:   "gamma",
			want:     "gamma beta gamma",
		},
		{
			name:      "replaces only first exact match when requested",
			existing:  "alpha beta alpha",
			oldStr:    "alpha",
			newStr:    "gamma",
			want:      "gamma beta alpha",
			firstOnly: true,
		},
		{
			name:     "errors when target is empty",
			existing: "ab",
			oldStr:   "",
			newStr:   "-",
			wantErr:  true,
		},
		{
			name:     "errors when target is missing",
			existing: "alpha beta",
			oldStr:   "gamma",
			newStr:   "delta",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := replacePageContent(tt.existing, tt.oldStr, tt.newStr, tt.firstOnly)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

package booksing

import (
	"reflect"
	"testing"
)

func TestNextShelveIcon(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    *ShelveIcon
		wantErr bool
	}{
		{
			name:    "invalid",
			arg:     "",
			want:    nil,
			wantErr: true,
		},
		{
			name: "simple",
			arg:  "star",
			want: &ShelveIcon{
				"star-fill",
				"",
			},
			wantErr: false,
		},
		{
			name: "rollover",
			arg:  "globe",
			want: &ShelveIcon{
				"star",
				"",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextShelveIcon(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextShelveIcon() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got[0], tt.want[0]) { // we only care about the icon, not the theme
				t.Errorf("NextShelveIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

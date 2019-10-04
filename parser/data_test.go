package parser

import (
	"reflect"
	"testing"
)

func TestEncodeBody(t *testing.T) {
	type args struct {
		event string
		body  []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "name 1", args: args{event: "event1", body: []byte("body1")},
			want: []byte("event1" + Delimiter + "body1")},
		{name: "name 2", args: args{event: "ev", body: []byte("bod")},
			want: []byte("ev" + Delimiter + "bod")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EncodeBody(tt.args.event, tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EncodeBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseData(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name  string
		args  args
		want1 string
		want2 []byte
		want3 error
	}{
		{name: "name 1", args: args{data: []byte("event1" + Delimiter + "body1")},
			want1: "event1", want2: []byte("body1"), want3: nil},
		{name: "name 2", args: args{data: []byte("ev" + Delimiter + "bod")},
			want1: "ev", want2: []byte("bod"), want3: nil},
		{name: "name 3", args: args{data: []byte("CLIENT_INIT" + Delimiter + "bod")},
			want1: "CLIENT_INIT", want2: []byte("bod"), want3: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ParseData(tt.args.data)
			if got != tt.want1 {
				t.Errorf("ParseData() got = %v, want %v", got, tt.want1)
			}
			if !reflect.DeepEqual(got1, tt.want2) {
				t.Errorf("ParseData() got1 = %v, want %v", got1, tt.want2)
			}
			if !reflect.DeepEqual(err, tt.want3) {
				t.Errorf("ParseData() err = %v, want %v", got1, tt.want2)
			}
		})
	}
}

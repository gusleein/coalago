package message

import (
	"testing"
)

func TestCoAPMessageOption_IsElective(t *testing.T) {
	type fields struct {
		Code  OptionCode
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"even", fields{2, nil}, true,
		},
		{
			"odd", fields{3, nil}, false,
		},
		{
			"zero", fields{0, nil}, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &CoAPMessageOption{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := o.IsElective(); got != tt.want {
				t.Errorf("CoAPMessageOption.IsElective() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCoAPMessageOption_IsCritical(t *testing.T) {
	type fields struct {
		Code  OptionCode
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"even", fields{2, nil}, false,
		},
		{
			"odd", fields{3, nil}, true,
		},
		{
			"zero", fields{0, nil}, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &CoAPMessageOption{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := o.IsCritical(); got != tt.want {
				t.Errorf("CoAPMessageOption.IsCritical() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCoAPMessageOption_StringValue(t *testing.T) {
	str := "hello"

	type fields struct {
		Code  OptionCode
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"string", fields{1, str}, str,
		},
		{
			"int", fields{1, 42}, "",
		},
		{
			"struct", fields{1, struct{}{}}, "",
		},
		{
			"nil", fields{1, nil}, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &CoAPMessageOption{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := o.StringValue(); got != tt.want {
				t.Errorf("CoAPMessageOption.StringValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCoAPMessageOption_IntValue(t *testing.T) {
	type fields struct {
		Code  OptionCode
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			"int", fields{1, int(42)}, 42,
		},
		{
			"int8", fields{1, int8(42)}, 42,
		},
		{
			"int16", fields{1, int16(42)}, 42,
		},
		{
			"int32", fields{1, int32(42)}, 42,
		},
		{
			"uint", fields{1, uint(42)}, 42,
		},
		{
			"uint8", fields{1, uint8(42)}, 42,
		},
		{
			"uint16", fields{1, uint16(42)}, 42,
		},
		{
			"uint32", fields{1, uint32(42)}, 42,
		},
		{
			"string", fields{1, string("42")}, 42,
		},
		{
			"string invalid", fields{1, string("hello!")}, 0,
		},
		{
			"struct", fields{1, struct{}{}}, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &CoAPMessageOption{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := o.IntValue(); got != tt.want {
				t.Errorf("CoAPMessageOption.IntValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCoAPMessageOption_IsRepeatableOption(t *testing.T) {
	type fields struct {
		Code  OptionCode
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"OptionIfMatch", fields{OptionIfMatch, nil}, true,
		},
		{
			"OptionEtag", fields{OptionEtag, nil}, true,
		},
		{
			"OptionLocationPath", fields{OptionLocationPath, nil}, true,
		},
		{
			"OptionURIPath", fields{OptionURIPath, nil}, true,
		},
		{
			"OptionURIQuery", fields{OptionURIQuery, nil}, true,
		},
		{
			"OptionLocationQuery", fields{OptionLocationQuery, nil}, true,
		},
		{
			"OptionURIPort", fields{OptionURIPort, nil}, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &CoAPMessageOption{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := opt.IsRepeatableOption(); got != tt.want {
				t.Errorf("CoAPMessageOption.IsRepeatableOption() = %v, want %v", got, tt.want)
			}
		})
	}
}

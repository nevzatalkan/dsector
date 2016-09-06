package ufwb

import (
	"github.com/kylelemons/godebug/pretty"
	"testing"
)

func TestExtendDifferentTypes(t *testing.T) {

	var tests = []struct {
		src interface{}
		dst interface{}
	}{
		{1, &Structure{}},
		{&Structure{}, 1},
		{&Number{}, &Structure{}},
		{[]Structure{}, &Structure{}},

		{Structure{}, &Structure{}},
	}

	for _, test := range tests {
		if err := extend(test.dst, test.src); err == nil {
			t.Errorf("extend(%T, %T) = nil want error", test.dst, test.src)
		}
	}
}

func TestExtends(t *testing.T) {
	src := &Structure{
		IdName: IdName{1, "struct", "better description"},
		Endian: BigEndian,
		Elements: []Element{
			&String{
				IdName: IdName{2, "string", ""},
				Type:   "zero-terminated",
			},
			&Number{
				IdName: IdName{3, "number", ""},
				Type:   "integer",
				Length: "4",
			},
			&Structure{
				IdName: IdName{4, "substruct", ""},
				Length: "prev.size",
			},
		},
	}

	dst := &Structure{
		IdName: IdName{11, "struct", ""},
		Elements: []Element{
			&String{
				IdName: IdName{12, "string", ""},
			},
			&Number{
				IdName: IdName{13, "number", ""},
			},
			&Structure{
				IdName: IdName{14, "substruct", ""},
			},
			&Number{
				IdName: IdName{15, "number2", ""},
				Length: "2",
			},
		},
	}

	want := &Structure{
		IdName: IdName{11, "struct", "better description"},
		Endian: BigEndian, // Copied from src
		Elements: []Element{
			&String{
				IdName: IdName{12, "string", ""},
				Type:   "zero-terminated", // Copied from src
			},
			&Number{
				IdName: IdName{13, "number", ""},
				Type:   "integer", // Copied from src
				Length: "4",       // Copied from src
			},
			&Structure{
				IdName: IdName{14, "substruct", ""},
				Length: "prev.size", // Copied from src
			},
			&Number{
				IdName: IdName{15, "number2", ""},
				Length: "2",
			},
		},
	}

	if err := extend(dst, src); err != nil {
		t.Errorf("extend(...) = %q want nil", err)
	}

	if diff := pretty.Compare(dst, want); diff != "" {
		t.Errorf("extend(...) dst = -got +want:\n%s", diff)
	}
}

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"go.yaml.in/yaml/v4"
)

func TestResolveType(t *testing.T) {
	for _, expr := range []string{
		"string", "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "byte", "rune", "any",
		"[]*string", "[4]int", "map[string][]bool", "(*int)",
	} {
		if _, err := resolveType(expr); err != nil {
			t.Errorf("%s: %v", expr, err)
		}
	}
	for _, expr := range []string{
		"", "unknown", "struct { X int }", "func()", "chan int",
		"map[[]int]int", "[-1]int", "[1+2]int", "[999999999999999999999]int",
		"[...]int", "[]", "pkg.Type",
	} {
		if _, err := resolveType(expr); err == nil {
			t.Errorf("accepted %q", expr)
		}
	}
}

func TestTypedErrors(t *testing.T) {
	for _, flags := range [][]string{
		{"-n"},
		{"-N"},
		{"-t"},
		{"-T"},
		{"-e"},
		{"-E"},
		{"-Y"},
		{"-j", "-n"},
		{"-j", "--from=node"},
	} {
		args := append([]string{"--type", "[]string"}, flags...)
		cmd := exec.Command(testBinary, args...)
		cmd.Stdin = strings.NewReader("[a]")
		if output, err := cmd.CombinedOutput(); err == nil {
			t.Errorf("%v unexpectedly succeeded: %s", args, output)
		}
	}
}

func TestTypedUTF16(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for _, tc := range []struct {
			name  string
			units []uint16
			want  string
		}{
			{"high then newline", []uint16{0xd800, 10}, ""},
			{"lone high", []uint16{0xd800}, ""},
			{"lone low", []uint16{0xdc00}, ""},
			{"high then high", []uint16{0xd800, 0xd800}, ""},
			{"valid pair", []uint16{0xd83d, 0xde00}, "😀"},
			{"replacement character", []uint16{0xfffd}, "�"},
		} {
			t.Run(fmt.Sprintf("%s/%s", order, tc.name), func(t *testing.T) {
				input := make([]byte, 2+2*len(tc.units))
				order.PutUint16(input, 0xfeff)
				for i, unit := range tc.units {
					order.PutUint16(input[2+2*i:], unit)
				}
				var value string
				nativeErr := yaml.Unmarshal(input, &value)
				if (nativeErr == nil) != (tc.want != "") {
					t.Fatalf("unexpected native result: %q, %v", value, nativeErr)
				}
				for _, mode := range []string{"-y", "-j"} {
					cmd := exec.Command(testBinary, "--type", "string", "--unmarshal", mode)
					cmd.Stdin = bytes.NewReader(input)
					output, err := cmd.Output()
					if (err == nil) != (nativeErr == nil) {
						t.Fatalf("%s: native error %v, CLI error %v, output %q",
							mode, nativeErr, err, output)
					}
					if err == nil {
						var got string
						if err := yaml.Unmarshal(output, &got); err != nil || got != tc.want {
							t.Fatalf("%s: output %q, error %v", mode, output, err)
						}
					}
				}
			})
		}
	}
}

func TestTypedDestinationIsolation(t *testing.T) {
	for _, name := range []string{"bool", "string", "any"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			typ, err := resolveType(name)
			if err != nil {
				t.Fatal(err)
			}
			dst := newDestination(typ)
			if err := loadDestination(typ, []byte("yes"), dst); err != nil {
				t.Fatal(err)
			}
			var want any = "yes"
			if name == "bool" {
				want = true
			}
			if got := reflect.ValueOf(dst).Elem().Interface(); got != want {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			generic := newDestination(nil)
			if err := loadDestination(nil, []byte("yes"), generic); err != nil {
				t.Fatal(err)
			}
			if got := *generic.(*any); got != "yes" {
				t.Fatalf("legacy Load returned %#v", got)
			}
		})
	}
}

func TestTypedUnicodeDocuments(t *testing.T) {
	input := "\"🧜‍♂️\"\n---\n\"é😀尾\""
	encodings := map[string][]byte{
		"UTF-8":     []byte(input),
		"UTF-8 BOM": []byte("\ufeff" + input),
	}
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		units := utf16.Encode([]rune("\ufeff" + input))
		data := make([]byte, 2*len(units))
		for i, unit := range units {
			order.PutUint16(data[2*i:], unit)
		}
		encodings[order.String()] = data
	}
	for name, input := range encodings {
		t.Run(name, func(t *testing.T) {
			for _, mode := range []string{"-j", "-y"} {
				cmd := exec.Command(testBinary, "--type", "string", "--unmarshal", mode)
				cmd.Stdin = bytes.NewReader(input)
				output, err := cmd.Output()
				if err != nil {
					t.Fatal(err)
				}
				decode := yaml.NewDecoder(bytes.NewReader(output)).Decode
				if mode == "-j" {
					decode = json.NewDecoder(bytes.NewReader(output)).Decode
				}
				for _, want := range []string{"🧜‍♂️", "é😀尾"} {
					var got string
					if err := decode(&got); err != nil || got != want {
						t.Fatalf("%s: got %q, want %q, error %v", mode, got, want, err)
					}
				}
				var extra any
				if err := decode(&extra); err != io.EOF {
					t.Fatalf("%s: expected EOF, got %v", mode, err)
				}
			}
		})
	}
}

func TestArrayLengthBound(t *testing.T) {
	expr := fmt.Sprintf("[%d]int", uint64(math.MaxInt)+1)
	if _, err := resolveType(expr); err == nil ||
		!strings.Contains(err.Error(), "exceeds int range") {
		t.Fatalf("%s: expected int range error, got %v", expr, err)
	}
}

func TestArrayLengthLiterals(t *testing.T) {
	for _, literal := range []string{"10", "0b1010", "0o12", "012", "0xa", "1_0"} {
		typ, err := resolveType("[" + literal + "]int")
		if err != nil {
			t.Fatal(err)
		}
		if typ.Len() != 10 {
			t.Errorf("%s: got length %d", literal, typ.Len())
		}
	}
	for _, literal := range []string{"18446744073709551616", "0x10000000000000000"} {
		if _, err := resolveType("[" + literal + "]int"); err == nil {
			t.Errorf("accepted oversized length %s", literal)
		}
	}
}

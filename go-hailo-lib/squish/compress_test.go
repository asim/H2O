package squish

import (
	"testing"
)

func Test1(t *testing.T) {
	in := "198634378235"

	out := Compress36(in)

	if out != "2j91s35n" {
		t.Errorf("Input string: %s compressed string: %s", in, out)
	}

	out = Uncompress36(out)

	if out != in {
		t.Errorf("Input string: %s cycled string: %s", in, out)
	}
}

func Test2(t *testing.T) {
	in := "MAD382488030102724608"

	out := in[:3] + Compress36(in[3:])

	if out != "MAD2wm4ovaevklc" {
		t.Errorf("Input string: %s compressed string: %s", in, out)
	}

	out = in[:3] + Uncompress36(out[3:])

	if out != in {
		t.Errorf("Input string: %s cycled string: %s", in, out)
	}
}

func Test3(t *testing.T) {
	in := "MAD382488030102724608"

	out := CompressTail36(3, in)

	if out != "MAD2wm4ovaevklc" {
		t.Errorf("Input string: %s (%d) compressed string: %s (%d)", in, len(in), out, len(out))
	}

	if len(out) > 18 {
		t.Errorf("Input string: %s compressed string: %s compressed string too long: %d", in, out, len(out))
	}

	out = UncompressTail36(3, out)

	if out != in {
		t.Errorf("Input string: %s cycled string: %s", in, out)
	}
}

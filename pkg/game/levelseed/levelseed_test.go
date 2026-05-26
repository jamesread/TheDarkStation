package levelseed

import "testing"

func TestFormat(t *testing.T) {
	if got := Format(42); got != "2A" {
		t.Fatalf("Format(42) = %q, want 2A", got)
	}
	if got := Format(1779797637817431329); got != "18B31C85962FF521" {
		t.Fatalf("Format(problem seed) = %q", got)
	}
}

func TestParse(t *testing.T) {
	seed, err := Parse("18B31C85962FF521")
	if err != nil {
		t.Fatal(err)
	}
	if seed != 1779797637817431329 {
		t.Fatalf("seed = %d", seed)
	}

	seed, err = Parse("0x2A")
	if err != nil {
		t.Fatal(err)
	}
	if seed != 42 {
		t.Fatalf("seed = %d", seed)
	}

	if _, err := Parse(""); err == nil {
		t.Fatal("expected error for empty seed")
	}
	if _, err := Parse("ZZ"); err == nil {
		t.Fatal("expected error for invalid seed")
	}
}

func TestParseRoundTrip(t *testing.T) {
	const original = int64(1779797637817431329)
	parsed, err := Parse(Format(original))
	if err != nil {
		t.Fatal(err)
	}
	if parsed != original {
		t.Fatalf("round trip = %d, want %d", parsed, original)
	}
}

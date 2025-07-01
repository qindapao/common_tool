package hex

import (
	"os"
	"testing"
)

const testHexStr = "0x1A2B3C\n"

func BenchmarkParseHexToUint64_64bit(b *testing.B) {
	for b.Loop() {
		_, err := ParseHexToUint64(testHexStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseHexToUint64_32bit(b *testing.B) {
	for b.Loop() {
		_, err := ParseHexToUint64(testHexStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadHexToUint64Ff_FileIO_64bit(b *testing.B) {
	tmp := createHexTempFile(b, testHexStr)
	defer os.Remove(tmp)

	b.ResetTimer()
	for b.Loop() {
		_, err := ReadHexToUint64Ff(tmp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadHexToUint64Ff_FileIO_32bit(b *testing.B) {
	tmp := createHexTempFile(b, testHexStr)
	defer os.Remove(tmp)

	b.ResetTimer()
	for b.Loop() {
		_, err := ReadHexToUint64Ff(tmp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createHexTempFile(b *testing.B, content string) string {
	b.Helper()
	f, err := os.CreateTemp("", "hex_bench_*.txt")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		b.Fatal(err)
	}
	f.Close()
	return f.Name()
}
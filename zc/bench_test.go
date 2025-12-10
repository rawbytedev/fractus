package zc

import (
	db "fractus/pkg/dbflat"
	"testing"
)

func benchFields(count int) []FieldValue {
	fields := make([]FieldValue, 0, count)
	for i := 0; i < count; i++ {
		tag := uint16(i + 1)
		fields = append(fields, FieldValue{Tag: tag, CompFlags: 0x8000, Payload: []byte("hello world payload data")})
	}
	return fields
}
func benchFieldsdb(count int) []db.FieldValue {
	fields := make([]db.FieldValue, 0, count)
	for i := 0; i < count; i++ {
		tag := uint16(i + 1)
		fields = append(fields, db.FieldValue{Tag: tag, CompFlags: 0x8000, Payload: []byte("hello world payload data")})
	}
	return fields
}

func Benchmark_GenTagWalk_zc(b *testing.B) {
	fields := benchFields(8)
	b.ReportAllocs()
	zc := NewZeroCopy()
	for i := 0; i < b.N; i++ {
		_ = zc.GenTagWalk(fields)
	}
}

func Benchmark_GenTagWalk_db(b *testing.B) {
	fields := benchFieldsdb(8)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = db.GenTagWalk(fields)
	}
}

func Benchmark_EncodeRecordTagWalk_zc(b *testing.B) {
	fields := benchFields(8)
	b.ReportAllocs()
	zc := NewZeroCopy()
	for i := 0; i < b.N; i++ {
		_, _ = zc.EncodeRecordTagWalk(fields)
	}
}

func Benchmark_EncodeRecordTagWalk_db(b *testing.B) {
	fields := benchFieldsdb(8)
	enc := db.Encoder{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = enc.EncodeRecordTagWorK(fields)
	}
}

func Benchmark_EncodeRecordHot_zc(b *testing.B) {
	fields := benchFields(8)
	hot := []uint16{1, 2, 3}
	zc := NewZeroCopy()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = zc.EncodeRecordHot(0xDEADBEEF, hot, fields)
	}
}

func Benchmark_EncodeRecordHot_db(b *testing.B) {
	fields := benchFieldsdb(8)
	hot := []uint16{1, 2, 3}
	enc := db.Encoder{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = enc.EncodeRecordFull(0xDEADBEEF, hot, fields)
	}
}

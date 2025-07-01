package bit

import "fmt"

type BitField struct {
	Name       string
	Start, Len byte
}

type FieldValue struct {
	BitField *BitField
	Value    uint64
}

func (f *BitField) Eval(val uint64) FieldValue {
	v := ExtractBits(val, f.Start, f.Len)
	return FieldValue{
		BitField: f, // 保留引用
		Value:    v,
	}
}

func (f *FieldValue) String() string {
	return f.BitField.Name + fmt.Sprintf("=0x%X [bits %d:%d]", f.Value, f.BitField.Start+f.BitField.Len-1, f.BitField.Start)
}

// 批量从多个字段提取值
//
//	fields := []*bit.BitField{
//	   {Name: "BaseClass", Start: 16, Len: 8},
//	   {Name: "SubClass",  Start: 8,  Len: 8},
//	   {Name: "ProgIF",    Start: 0,  Len: 8},
//	}
//
// reg := uint64(0x060400)
// values := bit.EvalAll(fields, reg)
//
//	for _, v := range values {
//	   fmt.Println(v)
//	}
//
// BaseClass=0x6 [bits 23:16]
// SubClass=0x4 [bits 15:8]
// ProgIF=0x0 [bits 7:0]
func EvalAll(fields []*BitField, val uint64) []FieldValue {
	var out []FieldValue
	for _, f := range fields {
		out = append(out, f.Eval(val))
	}
	return out
}

// 对齐的格式化输出
func FormatFieldValues(vals []FieldValue) string {
	maxNameLen := 0
	for _, v := range vals {
		if l := len(v.BitField.Name); l > maxNameLen {
			maxNameLen = l
		}
	}

	var out string
	for _, v := range vals {
		// 这里只是一个模板，需要%，所以要在前面打%转义
		format := fmt.Sprintf("%%-%ds = 0x%%-X [bits %%2d:%%d]\n", maxNameLen)
		out += fmt.Sprintf(format,
			v.BitField.Name,
			v.Value,
			v.BitField.Start+v.BitField.Len-1,
			v.BitField.Start,
		)
	}
	return out
}

// 转装字段的完整值
func PackFields(fields []FieldValue) uint64 {
	var out uint64
	for _, f := range fields {
		shifted := f.Value << f.BitField.Start
		out |= shifted
	}
	return out
}

// 寄存器读取接口
type Reader interface {
	Read(offset uint32, size byte) uint64
}

// 普通函数读取型
type FunctionReader func(offset uint32, size byte) uint64

func (f FunctionReader) Read(offset uint32, size byte) uint64 {
	return f(offset, size)
}

// 闭包读取型
type BoundReader struct {
	ReadFunc func() uint64 // 捕获上下文的闭包
}

func (r BoundReader) Read(_ uint32, _ byte) uint64 {
	return r.ReadFunc()
}

// // 共享函数型寄存器
// reg0 := RegisterDescriptor{
//     Name:   "REG0",
//     Offset: 0x00,
//     Size:   4,
//     Reader: FunctionReader(ReadMMIO), // 统一使用
// }

// // 闭包自带行为型寄存器
//
//	reg1 := RegisterDescriptor{
//	    Name:   "REG1",
//	    Size:   4,
//	    Reader: BoundReader{
//	        ReadFunc: func() uint64 {
//	            return mmio.Read32(0x10000000 + 0x04)
//	        },
//	    },
//	}
//
// 下面这种调用两种方式都可以，闭包的时候这两个参数忽略也不报错
// val := reg.Reader.Read(reg.Offset, reg.Size)
type RegisterDescriptor struct {
	Name   string
	Offset uint32      // 如果是配置空间 / MMIO 可用
	Size   byte        // 寄存器大小（单位：byte）
	Fields []*BitField // 所有字段
	Doc    string      // 文档注释 / 描述（可选）
	Reader Reader      // 读取函数
}

func (r *RegisterDescriptor) Eval(val uint64) []FieldValue {
	return EvalAll(r.Fields, val)
}

func (r *RegisterDescriptor) Format(val uint64) string {
	values := r.Eval(val)
	return FormatFieldValues(values)
}

// C 语言版本 ======================普通函数版本======================

// bitfield.h

// #ifndef BITFIELD_H
// #define BITFIELD_H

// #include <stdint.h>
// #include <stddef.h>

// // 位字段定义
// typedef struct {
//     const char* name;
//     uint8_t start;
//     uint8_t len;
// } BitField;

// // 字段值
// typedef struct {
//     const BitField* def; // 引用字段定义
//     uint64_t value;
// } FieldValue;

// // 提取字段值
// FieldValue eval_field(const BitField* field, uint64_t raw);

// // 批量提取字段值
// void eval_all(const BitField* fields, size_t count, uint64_t raw, FieldValue* out);

// // 格式化输出（到 stdout）
// void print_field_value(const FieldValue* fv);
// void print_all_fields(const FieldValue* vals, size_t count);

// // 将多个字段合成为一个值
// uint64_t pack_fields(const FieldValue* vals, size_t count);

// #endif // BITFIELD_H

// #include "bitfield.h"
// #include <stdio.h>

// static inline uint64_t extract_bits(uint64_t val, uint8_t start, uint8_t len) {
//     return (val >> start) & ((1ULL << len) - 1);
// }

// FieldValue eval_field(const BitField* field, uint64_t raw) {
//     FieldValue fv;
//     fv.def = field;
//     fv.value = extract_bits(raw, field->start, field->len);
//     return fv;
// }

// void eval_all(const BitField* fields, size_t count, uint64_t raw, FieldValue* out) {
//     for (size_t i = 0; i < count; ++i) {
//         out[i] = eval_field(&fields[i], raw);
//     }
// }

// void print_field_value(const FieldValue* fv) {
//     printf("%s = 0x%llX [bits %d:%d]\n",
//            fv->def->name,
//            fv->value,
//            fv->def->start + fv->def->len - 1,
//            fv->def->start);
// }

// void print_all_fields(const FieldValue* vals, size_t count) {
//     size_t max_len = 0;
//     for (size_t i = 0; i < count; ++i) {
//         size_t len = strlen(vals[i].def->name);
//         if (len > max_len) max_len = len;
//     }

//     for (size_t i = 0; i < count; ++i) {
//         const FieldValue* fv = &vals[i];
//         printf("%-*s = 0x%llX [bits %2d:%d]\n",
//                (int)max_len,
//                fv->def->name,
//                fv->value,
//                fv->def->start + fv->def->len - 1,
//                fv->def->start);
//     }
// }

// uint64_t pack_fields(const FieldValue* vals, size_t count) {
//     uint64_t out = 0;
//     for (size_t i = 0; i < count; ++i) {
//         const FieldValue* fv = &vals[i];
//         out |= (fv->value << fv->def->start);
//     }
//     return out;
// }
//

// 用法举例：

// #include "bitfield.h"

// int main() {
//     BitField fields[] = {
//         { "BaseClass", 16, 8 },
//         { "SubClass",  8, 8 },
//         { "ProgIF",    0, 8 },
//     };

//     uint64_t val = 0x060400;
//     FieldValue decoded[3];

//     eval_all(fields, 3, val, decoded);
//     print_all_fields(decoded, 3);

//     uint64_t reassembled = pack_fields(decoded, 3);
//     printf("\nReconstructed: 0x%06llX\n", reassembled);

//     return 0;
// }

// 输出类似：

// BaseClass = 0x6   [bits 23:16]
// SubClass  = 0x4   [bits 15:8]
// ProgIF    = 0x0   [bits  7:0]

// Reconstructed: 0x060400
//
// 宏版本 ===========================================轻量宏版本============
//
// bitfield.h
// #ifndef BITFIELD_H
// #define BITFIELD_H
//
// #include <stdint.h>
// #include <stddef.h>
//
// // === 类型定义 ===
//
// typedef struct {
//     const char* name;
//     uint8_t start;
//     uint8_t len;
// } BitField;
//
// typedef struct {
//     const BitField* def;
//     uint64_t value;
// } FieldValue;
//
// // === 宏定义（结构宏、安全无副作用） ===
//
// #define BITFIELD(name, start, len) \
//     { .name = name, .start = start, .len = len }
//
// // 如果将来有寄存器结构，可加：
// #define REGISTER(name, offset, size, fields) \
//     { name, offset, size, fields, sizeof(fields)/sizeof(fields[0]) }
//
// // === 函数接口 ===
//
// FieldValue eval_field(const BitField* field, uint64_t raw);
// void eval_all(const BitField* fields, size_t count, uint64_t raw, FieldValue* out);
//
// void print_field_value(const FieldValue* fv);
// void print_all_fields(const FieldValue* vals, size_t count);
//
// uint64_t pack_fields(const FieldValue* vals, size_t count);
//
// #endif // BITFIELD_H
//
//
// bitfield.c
//
// #include "bitfield.h"
// #include <stdio.h>
// #include <string.h>
//
// static inline uint64_t extract_bits(uint64_t val, uint8_t start, uint8_t len) {
//     return (val >> start) & ((1ULL << len) - 1);
// }
//
// FieldValue eval_field(const BitField* field, uint64_t raw) {
//     FieldValue fv;
//     fv.def = field;
//     fv.value = extract_bits(raw, field->start, field->len);
//     return fv;
// }
//
// void eval_all(const BitField* fields, size_t count, uint64_t raw, FieldValue* out) {
//     for (size_t i = 0; i < count; ++i) {
//         out[i] = eval_field(&fields[i], raw);
//     }
// }
//
// void print_field_value(const FieldValue* fv) {
//     printf("%s = 0x%llX [bits %d:%d]\n",
//            fv->def->name,
//            fv->value,
//            fv->def->start + fv->def->len - 1,
//            fv->def->start);
// }
//
// void print_all_fields(const FieldValue* vals, size_t count) {
//     size_t max_len = 0;
//     for (size_t i = 0; i < count; ++i) {
//         size_t len = strlen(vals[i].def->name);
//         if (len > max_len) max_len = len;
//     }
//
//     for (size_t i = 0; i < count; ++i) {
//         const FieldValue* fv = &vals[i];
//         printf("%-*s = 0x%llX [bits %2d:%d]\n",
//                (int)max_len,
//                fv->def->name,
//                fv->value,
//                fv->def->start + fv->def->len - 1,
//                fv->def->start);
//     }
// }
//
// uint64_t pack_fields(const FieldValue* vals, size_t count) {
//     uint64_t out = 0;
//     for (size_t i = 0; i < count; ++i) {
//         const FieldValue* fv = &vals[i];
//         out |= (fv->value << fv->def->start);
//     }
//     return out;
// }
//
// ============================================内核版本======================
// bitfield.h
// #ifndef BITFIELD_H
// #define BITFIELD_H

// #include <linux/types.h>
// #include <linux/kernel.h>
// #include <linux/printk.h>
// #include <linux/string.h>

// // === 结构定义 ===

// struct bit_field {
//     const char *name;
//     u8 start;
//     u8 len;
// };

// struct field_value {
//     const struct bit_field *def;
//     u64 value;
// };

// // === 安全结构初始化宏（无副作用） ===

// #define BIT_FIELD_INIT(_name, _start, _len) \
//     { .name = (_name), .start = (_start), .len = (_len) }

// // === 函数接口 ===

// static inline u64 extract_bits(u64 val, u8 start, u8 len)
// {
//     return (val >> start) & ((1ULL << len) - 1);
// }

// static inline struct field_value eval_field(const struct bit_field *f, u64 raw)
// {
//     struct field_value out = {
//         .def = f,
//         .value = extract_bits(raw, f->start, f->len),
//     };
//     return out;
// }

// static inline void eval_fields(const struct bit_field *fields,
//                    size_t count, u64 raw,
//                    struct field_value *out)
// {
//     size_t i;
//     for (i = 0; i < count; ++i)
//         out[i] = eval_field(&fields[i], raw);
// }

// static inline u64 pack_fields(const struct field_value *vals, size_t count)
// {
//     u64 acc = 0;
//     size_t i;
//     for (i = 0; i < count; ++i) {
//         const struct field_value *fv = &vals[i];
//         acc |= (fv->value << fv->def->start);
//     }
//     return acc;
// }

// static inline void print_field_value(const struct field_value *fv)
// {
//     pr_info("%s = 0x%llx [bits %u:%u]\n",
//         fv->def->name,
//         fv->value,
//         fv->def->start + fv->def->len - 1,
//         fv->def->start);
// }

// static inline void print_all_fields(const struct field_value *vals, size_t count)
// {
//     size_t i;
//     for (i = 0; i < count; ++i)
//         print_field_value(&vals[i]);
// }

// #endif /* BITFIELD_H */

// 使用示例：

// #include <linux/init.h>
// #include <linux/module.h>
// #include "bitfield.h"

// static int __init bitfield_demo_init(void)
// {
//     struct bit_field pci_fields[] = {
//         BIT_FIELD_INIT("BaseClass", 16, 8),
//         BIT_FIELD_INIT("SubClass", 8, 8),
//         BIT_FIELD_INIT("ProgIF", 0, 8),
//     };

//     u64 reg_val = 0x060400;
//     struct field_value results[ARRAY_SIZE(pci_fields)];

//     eval_fields(pci_fields, ARRAY_SIZE(pci_fields), reg_val, results);
//     print_all_fields(results, ARRAY_SIZE(results));

//     pr_info("Reconstructed: 0x%llx\n", pack_fields(results, ARRAY_SIZE(results)));

//     return 0;
// }

// static void __exit bitfield_demo_exit(void)
// {
//     pr_info("Bitfield demo unloaded.\n");
// }

// module_init(bitfield_demo_init);
// module_exit(bitfield_demo_exit);

// MODULE_LICENSE("GPL");
// MODULE_AUTHOR("你老铁");
// MODULE_DESCRIPTION("位字段解析：内核空间安全版示范");
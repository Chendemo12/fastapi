package types

// Ordered 是匹配任何有序类型的类型约束.
// An ordered type is one that supports the <, <=, >, and >= operators.
type Ordered interface {
	~int | ~uint | ~float64 | ~string
}

// ComparableHash is a type constraint that matches all
// comparable types with a Hash method.
type ComparableHash interface {
	comparable
	Hash() uintptr
}

// ImpossibleConstraint is a type constraint that no type can satisfy,
// because slice types are not comparable.
type ImpossibleConstraint interface {
	comparable
	[]int
}

type Number interface {
	~float32 | ~float64
}

type Integer interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~int | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint
}

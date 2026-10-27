package functions

func Add(x, y int) int         { return x + y }
func Effect(any)               {}
func Multiple() (int, string)  { return 1, "one" }
func Generic[T any](value T) T { return value }

# Go for Java Developers — as used in this project

Every Go construct that appears in the code, next to its closest Java
equivalent. Entries are added as new syntax shows up. The "Where" column
points to the first place it's used.

## Program structure

| Go | Java | Notes | Where |
|---|---|---|---|
| `package chunker` (one per folder) | `package com.x.chunker;` | A Go package is a *directory*. All files in it share one namespace, with no per-file class. | `rolling.go` |
| `Roll` (uppercase) / `roll` (lowercase) | `public` / package-private | Capitalization **is** visibility. There's no `private`, `protected` or `public` keyword. | everywhere |
| `// Package chunker ...` above `package` | Javadoc on `package-info.java` | `go doc` shows it. | `rolling.go` |
| `import ("bufio"; "io")` | `import java.io.*;` | Unused imports are a **compile error** in Go. | `chunker.go` |

## Types, structs and methods

| Go | Java | Notes | Where |
|---|---|---|---|
| `type RollingHash struct { hash uint64 }` | `class RollingHash { long hash; }` | Data only. Methods are declared separately, and there's no inheritance. | `rolling.go` |
| `func (r *RollingHash) Roll(b byte) uint64` | `long roll(byte b) { ... }` (instance method) | `r` is the explicit `this`. `*` means pointer receiver, so the method can modify the struct. | `rolling.go` |
| `func NewRollingHash(w int) *RollingHash` | constructor `new RollingHash(w)` | Go has no constructors. A `NewX` function that returns `*X` is the convention. | `rolling.go` |
| `&RollingHash{base: b}` | `new RollingHash()` plus setters | A struct literal with named fields. Unset fields get their *zero value* (0, `""`, `nil`, `false`). | `rolling.go` |
| `type Hash [32]byte` | `final class Hash { byte[] bytes; }` | A new named type built from an existing one. Methods can be attached to it even though it isn't a struct. | `store.go` |
| `func (h Hash) String() string` | `@Override String toString()` | Implements the `fmt.Stringer` interface, so `%s` and `fmt.Println` use it automatically. | `store.go` |

## Values, arrays and slices

| Go | Java | Notes | Where |
|---|---|---|---|
| `[32]byte` (array) | `byte[32]` | **Different!** A Go array is a *value*: assigning it copies all 32 bytes, and `==` compares contents. In Java, `byte[]` is a reference and `==` compares identity. That's why `Hash` can be compared with `!=` and used as a map key. | `store.go` |
| `[]byte` (slice) | `byte[]` / `ArrayList<Byte>` | A view into an array: (pointer, length, capacity). Passing a slice does **not** copy the bytes. | everywhere |
| `make([]byte, 0, 64*1024)` | `new ArrayList<>(65536)` | Length 0, pre-allocated capacity. | `chunker.go` |
| `chunk = append(chunk, b)` | `list.add(b)` | You must reassign. `append` may return a new backing array. | `chunker.go` |
| `h[:]` | `Arrays.copyOf(...)` (but no copy) | Slices an array into a slice that shares its memory. | `store.go` |
| `uint64` | `long` (signed only) | Java has no unsigned types. Both wrap around on overflow (two's complement), so the hash maths is the same. | `rolling.go` |
| `h >> shift` on a `uint64` | `h >>> shift` | On an unsigned type, `>>` is a logical shift (fills with zeros), like Java's `>>>`. | `chunker.go` |
| `const defaultBase uint64 = 0x1...` | `static final long DEFAULT_BASE` | | `rolling.go` |
| `var DefaultConfig = Config{...}` | `public static final Config DEFAULT` | Package-level variable. Go has no `final` for variables, so any code can reassign it. `const` only works for numbers, strings and booleans, not structs. | `chunker.go` |
| `0o755` | none (Java uses `PosixFilePermissions`) | An octal literal for Unix file permissions. | `store.go` |

## Errors (Go has no exceptions)

| Go | Java | Notes | Where |
|---|---|---|---|
| `func Next() ([]byte, error)` | `byte[] next() throws IOException` | Errors are ordinary **return values**. Functions can return several values. | `chunker.go` |
| `if err != nil { return nil, err }` | `throw` / letting an exception propagate | Propagation is explicit at every level, and the compiler forces you to handle the second return value or ignore it with `_`. | `chunker.go` |
| `io.EOF` | `read()` returning `-1` | A predefined error value meaning "finished normally". | `chunker.go` |
| `var ErrCorrupt = errors.New("...")` | `class CorruptException extends Exception` | A *sentinel* error: one shared value that callers compare against. | `store.go` |
| `fmt.Errorf("%w: %s", ErrCorrupt, h)` | `new CorruptException(msg, cause)` | `%w` *wraps*: the new error carries extra text but still "is" `ErrCorrupt`. | `store.go` |
| `errors.Is(err, fs.ErrNotExist)` | `catch (NoSuchFileException e)` / `instanceof` | Walks the wrap chain looking for a match. Prefer it over `err == X`. | `store.go` |
| `if err := f(); err != nil {...}` | none | Declares `err` scoped to just that `if`. | `store.go` |

## Interfaces

| Go | Java | Notes | Where |
|---|---|---|---|
| `io.Reader` (one method: `Read([]byte) (int, error)`) | `java.io.InputStream` | | `chunker.go` |
| A type "implements" by having the methods | `class X implements Reader` | **Implicit.** There's no `implements` keyword. Any type with a matching `Read` method *is* an `io.Reader`, even one written before the interface existed. | `chunker.go` |
| `bufio.NewReader(r)` | `new BufferedInputStream(in)` | Wraps a reader with a buffer, so `ReadByte` doesn't make a system call every time. | `chunker.go` |
| `bytes.NewReader(data)` | `new ByteArrayInputStream(data)` | An in-memory reader, handy in tests. | `chunker_test.go` |

## Maps, loops and control flow

| Go | Java | Notes | Where |
|---|---|---|---|
| `map[string]bool{}` / `make(map[string]bool)` | `new HashMap<String, Boolean>()` | A missing key returns the zero value (`false`), not `null`. | `chunker_test.go` |
| `seen[string(chunk)]` | none | A slice can't be a map key. Converting it to a `string` copies the bytes into a comparable key. Java has the same trap: `byte[]` keys use identity `equals`. | `chunker_test.go` |
| `for i, b := range data {}` | `for (int i = 0; ...) { byte b = data[i]; }` | `range` yields the index and the value. Use `_` for the one you don't need. | `rolling_test.go` |
| `for { ... }` | `while (true) { ... }` | Go has only `for`. It covers `while` and `do/while`. | `chunker.go` |
| `min(a, b)`, `max(a, b)` | `Math.min`, `Math.max` | Built-in functions since Go 1.21. | `chunker_test.go` |
| `x := 5` | `var x = 5;` | Declare and infer the type. Only allowed inside functions. | everywhere |

## Testing

| Go | Java (JUnit 5) | Notes | Where |
|---|---|---|---|
| `foo_test.go`, `func TestX(t *testing.T)` | `@Test void x()` | Found by naming convention, with no annotations. | all `_test.go` |
| `t.Fatalf("got %d", n)` | `fail(...)` / `assertEquals` | No assertion library. You write the `if` yourself. | all tests |
| `t.Logf(...)` | logger output | Shown only with `go test -v` or when the test fails. | `chunker_test.go` |
| `t.Helper()` | none | Failures are reported at the caller's line. | `chunker_test.go` |
| `t.TempDir()` | `@TempDir Path dir` | Created per test and deleted automatically afterwards. | `store_test.go` |
| `go test -race ./...` | none | Instruments the code to detect data races between goroutines at runtime. | — |

# Plaid

Plaid is a basic scripting language with closures and an optional typing. It was built as an exercise to learn more about language implementation.

## Examples

The canonical example:

    print "hello world"

An example of the closures in action:

    let newCounter := fn (n: Int): () => Int {
        n := n - 1
        return fn (): Int {
            n := n + 1
            return n
        }
    }

    let c1 := newCounter(1)
    let c2 := newCounter(10)

    print c1() # prints "1"
    print c1() # prints "2"

    print c2() # prints "10"
    print c2() # prints "11"

    print c1() # prints "3"

## TODO

Plaid is still unstable and prone to breaking changes. Upcoming changes (in no particular order) include:

+ An optional type system similar to [Dart](https://www.dartlang.org/articles/language/optional-types)
    + New syntax for declaring a typed-function's return type
+ Support for Swift-style string interpolation, ex: `"\(name) hello"`
+ A module import/export system

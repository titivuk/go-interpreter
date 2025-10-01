### Chapter 7: Functions

- function is constant value and `code.OpConstant` is used
- function is a sequence of instructions and `*object.CompiledFunction` represents that sequence
- `code.OpCall` to tell the VM to start executing the `*object.CompiledFunction` sitting on
top of the stack
- `code.OpReturnValue` to tell the VM to return the value on top of the stack to the calling
context and to resume execution there
- `code.OpReturn`, which is similar to code.OpReturnValue, except that there is no explicit
value to return but an implicit `vm.Null`

#### Insturctions scopes

If we were to simply call the compiler’s Compile method with the Body of the `*ast.FunctionLiteral` at hand, we’d end up with the resulting instructions being entangled with the instructions of the main program.

The solution is to introduce `CompilationScope`. Instead of using a single slice and the two separate fields `lastInstruction` and `previousInstruciton` to keep track of emitted instructions, we bundle them together in a `CompilationScope` and use stack of `CompilationScope`s.

When we parse `*ast.FunctionLiteral` body, we enter new scope. After parsing is finished we get the resulting instrucitons, wrap them into `*object.CompiledFunction`, the function to constant pool and emit `code.OpConstant`
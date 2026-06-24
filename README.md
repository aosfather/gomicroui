# gomicroui
microui go port
[简体中文](README_CN.md) | English

**Zero lines of code were written by humans. This project is 100% AI-generated."**

microui-> https://github.com/rxi/microui
# about
gomicroui is an experimental project — and a pretty fun one at that.​ I didn't write a single line of code for it myself; the entire codebase was produced by two agents, CodeBuddy and Trae. What I did was evaluate microui, the C library it's based on: it's written in C with no rendering backend bound to it, meaning you can get a working GUI system as long as you implement drawing a rectangle and text.

# progress
## The whole process turned out to be quite dramatic.​

I looked into microui and saw that it was only about a thousand lines of code, with zero dependencies outside the C standard library. Based on how capable AI coding tools have become lately, I figured there was a good chance the port would succeed—and even if it failed, jumping in to fix things myself wouldn’t be too much work.

So I grabbed the masterbranch, handed it to CodeBuddy, and gave it a simple task: port this project to Go.

What followed was oddly satisfying—CodeBuddy paused as if thinking, drafted a plan, then started listing files and reading through the code. It even suggested splitting things into modules like inputand controls, which I genuinely appreciated. Then it just went ahead and started converting everything.

## Then things got interesting.​

CodeBuddy finished translating the code and even started writing tests. There were some compilation errors, but luckily it managed to resolve them—for example, Go and C files can’t sit side by side, or cgokicks in and breaks the build (since we’re rewriting this purely in Go). Somehow, it ironed those out and got the tests running.

If you think that’s where the story ends, you’ve been fooled.

This is a GUI framework, so I told CodeBuddy to port the demo to Go as well. It dutifully generated the corresponding code under demo/—but then trouble started. The demo relied on SDL2, and CodeBuddy hallucinated libraries that don’t actually exist in Go. I nudged it to use a real GPU library instead. It tried hard, really—but after two attempts, it still couldn’t make it work.

And then, the final blow: my quota ran out.

CodeBuddy stopped responding to anything I said. That’s exactly why this project ended up involving two AI agents.

## As the saying goes, “listen to advice and you won’t go hungry.”​

When I switched to Trae, I made sure to explicitly tell it to use actual Go libraries​ for the demo instead of hallucinated ones. Trae accepted the task—after a long wait in the queue (the price of free stuff). It fumbled a few times, but eventually… the code compiled.

Of course, victory was short‑lived.

On launch, it crashed with a pointer error. I fed the error message back to Trae—and braced for another long wait. But to my surprise, it fixed the bug, muttering to itself about what went wrong and patching things up. Finally, it ran.

Well, kind of—it flickered like crazy and hurt my eyes.

I reported both issues: flickering and broken click handling. Queue again. More fiddling. And then—shockingly—it worked. No flicker. Correct clicks. Buttons responded, background colors changed, tree views expanded smoothly.

Bingo.​ Success. The Go port of microui was alive.

## To sum it up:​

CodeBuddy technicallytook my advice, but it had its own agenda. I suspect it was secretly inflating the context window on purpose… though I have no proof. (That’s a joke—mostly.)

Trae, on the other hand, came in later and just got it done. It cleanly accepted my suggestion: skip the SDL API translation entirely and reimplement the demo directly using Go’s GPU libraries. No drama, no detours—just execution.

## summary
In this project, I acted more like an architect guiding engineers through a migration.​ My role wasn't to write the code, but to identify issues and make the right technical decisions.

The two AI agents behaved just like human engineers with different personalities: some accept suggestions readily, others stick to their own logic. For me, this wasn't "vibe coding"—it was simply collaboration. The process of working with these AIs mirrors exactly how I work with human developers.
![gomicroui Demo](./demo.jpg)
# usage
This project is an experimental project that used AI to complete the conversion from C language to Go language, replaced the underlying implementation in the process, and successfully made the demo run normally.This project is an experimental project that used AI to complete the conversion from C language to Go language, replaced the underlying implementation in the process, and successfully made the demo run normally.

# example
```go
import (
	"microui"
)

```

#Qwen3 Coder
- Really fast
- Very good. One shotted the wildcard support for enabled tools and support for base url in env for openai
- Hesitated to use the zed built in edit tool but rather gave the code. Used the tool when nudged.
- One shotted support for anthropic on AWS Bedrock
#Gemini 2.5 Pro
- Works pretty well
- Wrote high level outline first with comments rather than trying to implement everything in one go
- Fleshing out the details with the available comments worked reasonably well
- Even this makes basis errors like not using &c instead of c when a assignment to a pointer
- Did not one shot implementation of Anthropic support (direct, not Bedrock)
  - Was overall right but there were improper struct types and fields
  - Suspecting due to mismatch between current version of the sdk and what the mpdel assumed
#GPT-5
- Seems to work similar to Gemini 2.5 Pro
- Hit the limits so did not do too much
#GPT-5 Mini
- Not very good
- Used some random guy's fork or implementation of openapi go sdk rather than use the official one
- Updated once explictly asked to
- Tried to fix errors by attempting to edit the sdk code rather than the faulty code it wrote
# Anthropic Claude on AWS Bedrock
- Request throttling issues
- Slow response
#General Observations
- Models seems to really get "confused" when there are multiple popular libraries for a solution, like the case of the openai go sdk, which have different ways of doing the same thing.
  - Mixes up things from both implementations
  - Many circular errror fixes, one fix gets back and earlier one
  - Suspecting that the point Yann LeCun had about doing the token generation in an abstract space rather than language space may help here. ToDo: find exactly what he said
  - Same issue when a library has breaking changes across versions.
- Code refactoring seems to work better if it is sufficiently composed into small functions. A very long function causes "confusion"
- I was able to generate ACP support in compell by vibe coding in Zed. Gemini 2.5 Pro, Claude and GPT-5 all had rate limit issues. Qwen was working fine but the code it generated did not work as expected. I tried switching the LLMS inbetween a number of times so not sure which one did how much of what I eneded up with.
  - Human still needed for integrations between systems.
  - Models do not at the moment have the intuition to troubleshoot issues by stepping out of the documentation and just printing out some traces  to see what is happening during execution.
  - Since ACP would have compell run by Zed I made Qwen write out traces to a file. It helped in the troubleshooting process.
  - Got into a real mess when trying to fix integration with Zed
  - There were many issues with the code such as incorrect types - type of ID was assigned to result and that of result to ID for example
  - Did not have any type checks and conversions for `any` type variables. Directly tried to find length.
    - The length check may have been a issue introduced by a vibe code attempt to fix the integration issue
    - I overdid the whole type check etc when I could have just json marshal->unmashal to turn the any to struct. Fixed manually but maybe the agent may have noticed it if I asked to review.
  - These may have been fixable by the agent with some retries but the issue that made me spend a lot of time investigating and tracing was how the agent did not complete initialization when integrated with Zed.
    - In ui it said initializing, in logs it seemed like the agent was waiting for some input from Zed which was not coming through
    - The issue was how the JSON RPC response was being writtent to stdout. It was missing a newline in the end which caused Zed to expect more data and wait for it indefinitely. Compell on the other hand was waiting for Zed to send a message which did not come as Zed was waiting.
    - It does not seem to me that this one would have been found by any coding assistant with any tools today. There is no error to work from. Just two programs waiting on each other.
- ACP implementation was missing tool calling. Had to look into trace to identify the issue. It was fixed by Claude Opus 4.1

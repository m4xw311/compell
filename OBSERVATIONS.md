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
#General Observations
- Models seems to really get "confused" when there are multiple popular libraries for a solution, like the case of the openai go sdk, which have different ways of doing the same thing.
  - Mixes up things from both implementations
  - Many circular errror fixes, one fix gets back and earlier one
  - Suspecting that the point Yann LeCun had about doing the token generation in an abstract space rather than language space may help here. ToDo: find exactly what he said
  - Same issue when a library has breaking changes across versions.
- Code refactoring seems to work better if it is sufficiently composed into small functions. A very long function causes "confusion"

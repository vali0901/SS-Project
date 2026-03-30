# Static analysis

* `cppcheck` results

```console
$ cppcheck --enable=all --inconclusive --std=c++11 src/main.cpp
Checking src/main.cpp: CAMERA_MODEL_AI_THINKER=CAMERA_MODEL_AI_THINKER...
src/main.cpp:189:37: style: C-style pointer casting [cstyleCast]
    if (client.publish(TOPIC_IMAGE, (const uint8_t*)fb->buf, fb->len)) {
                                    ^
src/main.cpp:156:6: style: The function 'setup' is never used. [unusedFunction]
void setup() {
     ^
```

* `bandit` results

```console
$ bandit -l $(find . -name "*.py")
Run started:2026-03-30 18:14:32.841470+00:00

Test results:
        No issues identified.

Code scanned:
        Total lines of code: 73
        Total lines skipped (#nosec): 0

Run metrics:
        Total issues (by severity):
                Undefined: 0
                Low: 0
                Medium: 0
                High: 0
        Total issues (by confidence):
                Undefined: 0
                Low: 0
                Medium: 0
                High: 0
Files skipped (0):
```

I guess we're too good for the game.

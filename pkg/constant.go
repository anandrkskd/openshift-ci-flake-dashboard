package pkg

const ansiTime = `\(\d+\.\d+s\)`
const ansiPrefix = `---\s+FAIL:\s+kuttl/harness/`
const ansiCommon = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansi [4]string = [4]string{ansiTime, ansiPrefix, ansiCommon}

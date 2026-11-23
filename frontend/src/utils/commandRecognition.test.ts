import { describe, expect, it } from 'vitest'
import { isShellCommand } from './commandRecognition'

describe('isShellCommand command catalog', () => {
  it.each([
    ['POSIX shell', 'bash'],
    ['shell builtin', 'export'],
    ['coreutils', 'ls'],
    ['text tool', 'awk'],
    ['process tool', 'systemctl'],
    ['network tool', 'ss'],
    ['archive tool', 'tar'],
    ['Linux package manager', 'apt-get'],
    ['macOS tool', 'launchctl'],
    ['Android tool', 'adb'],
    ['Android shell tool', 'getprop'],
    ['embedded tool', 'busybox'],
    ['Windows CMD', 'ipconfig'],
    ['Windows system tool', 'reg'],
    ['PowerShell cmdlet', 'Get-ChildItem'],
    ['PowerShell custom verb-noun command', 'Invoke-CustomDeployment'],
    ['version control', 'git'],
    ['JavaScript tool', 'pnpm'],
    ['Python tool', 'pipenv'],
    ['Go tool', 'gofmt'],
    ['Rust tool', 'cargo'],
    ['database client', 'sqlite3'],
    ['container tool', 'docker'],
    ['orchestration tool', 'kubectl'],
    ['infrastructure tool', 'terraform'],
    ['cloud tool', 'gcloud'],
  ])('recognizes %s commands', (_category, command) => {
    expect(isShellCommand(command)).toBe(true)
  })

  it.each([
    'LS -la',
    'python3.11 --version',
    'gcc-13 main.c',
    'custom-tool.exe /?',
    'custom-script.ps1',
    'Get-CustomWidget',
  ])('recognizes aliases, versions, suffixes and case variants: %s', (command) => {
    expect(isShellCommand(command)).toBe(true)
  })
})

describe('isShellCommand shell syntax', () => {
  it.each([
    '/usr/bin/python3 script.py',
    './deploy.sh',
    '../private-custom-script',
    'C:\\Windows\\System32\\cmd.exe /c dir',
    'sudo -u root rm -rf /',
    'doas custombinary',
    'env -i custombinary',
    'time custombinary',
    'nohup custombinary',
    'xargs custombinary',
    'busybox custom-applet',
    'cmd /c custombinary',
    'powershell -Command Get-ChildItem',
    'FOO=bar custombinary',
    'FOO=bar',
    'custombinary --help',
    'custombinary && othercustom',
    'custombinary | othercustom',
    'custombinary > output.txt',
    'custombinary 2>&1',
    'echo $(custombinary)',
    'echo `custombinary`',
    '"ls" -la',
    "'echo' hello",
  ])('recognizes command syntax: %s', (command) => {
    expect(isShellCommand(command)).toBe(true)
  })
})

describe('isShellCommand message boundary', () => {
  it.each([
    'hello',
    'hello world',
    'custombinary',
    '你好，世界！',
    'hello, world!',
    'I really like this place.',
    'lovely ✨ website',
    '2 + 2 = 4',
    'email me at test@example.com',
    'https://example.com/docs',
    '"hello; world"',
    'hello\\;world',
    'unclosed " quote',
  ])('allows ordinary text: %s', (message) => {
    expect(isShellCommand(message)).toBe(false)
  })

  it('handles malformed and long input without throwing', () => {
    expect(() => isShellCommand('unclosed ` substitution')).not.toThrow()
    expect(isShellCommand('hello '.repeat(1000))).toBe(false)
    expect(isShellCommand('   ')).toBe(false)
  })
})

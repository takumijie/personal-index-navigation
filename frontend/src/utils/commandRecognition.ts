const COMMAND_GROUPS = [
  // POSIX shells, builtins and keywords
  `sh bash zsh fish ksh csh tcsh dash ash hush alias bg cd command eval exec exit export fg jobs
   read readonly return set shift source trap umask unalias unset wait local typeset declare let
   test time [ . : true false if then else elif fi for while until do done case esac function select in`,
  // GNU coreutils and common file/text tools
  `arch b2sum base32 base64 basename basenc cat chcon chgrp chmod chown chroot cksum comm cp csplit
   cut date dd df dir dircolors dirname du echo env expand expr factor fmt fold head hostid hostname
   id install join link ln logname ls md5sum mkdir mkfifo mknod mktemp mv nice nl nohup nproc numfmt
   od paste pathchk pinky pr printenv printf ptx pwd readlink realpath rm rmdir runcon seq sha1sum
   sha224sum sha256sum sha384sum sha512sum shred shuf sleep sort split stat stdbuf stty sum sync tac
   tail tee timeout touch tr truncate tsort tty uname unexpand uniq unlink uptime users vdir wc who
   whoami yes awk sed grep egrep fgrep rg jq yq less more vim vi nano emacs ed ex patch diff diff3
   cmp file strings xargs iconv man info apropos whatis groff mandoc which whereis`,
  // util-linux, process, disk and system administration
  `agetty blkid blockdev cal chcpu chfn chrt chsh col colcrt colrm column dmesg eject fallocate
   fdformat fdisk findfs flock fsck fsfreeze fstrim getopt hexdump hwclock ionice ipcrm ipcs isosize
   last lastlog ldattach logger login look losetup lsblk lscpu lsfd lsipc lslocks lslogins lsmem lsns
   mesg mkswap mount mountpoint namei nsenter partx pivot_root prlimit raw readprofile rename renice
   resizepart rev rtcwake script scriptlive scriptreplay setarch setsid sfdisk su sulogin swaplabel
   swapoff swapon switch_root taskset umount unshare utmpdump uuidgen wall wdctl wipefs free pgrep
   pidof pmap ps pkill pwdx slabtop top vmstat w watch systemctl journalctl service reboot poweroff
   shutdown halt init telinit timedatectl localectl hostnamectl loginctl systemd-run dbus-send modprobe
   insmod rmmod lsmod lsof strace ltrace gdb perf crontab at batch useradd userdel usermod groupadd
   groupdel groupmod passwd chage newgrp sudo sudoedit doas visudo sysctl`,
  // Networking and remote access
  `arp arping curl dig ftp host ifconfig ip ip6tables iptables iw iwconfig mtr nc netcat netstat
   nslookup nmap ping ping6 route scp sftp ssh ssh-add ssh-agent ssh-keygen ssh-copy-id ss telnet
   traceroute tracepath wget socat tcpdump ethtool resolvectl networkctl nmcli openssl`,
  // Archives and compression
  `tar gzip gunzip bzip2 bunzip2 xz unxz zstd unzstd zip unzip 7z 7za rar unrar cpio lzma unlzma`,
  // Unix, Linux and macOS package managers
  `apt apt-get apt-cache aptitude dpkg dpkg-query snap flatpak dnf yum rpm zypper pacman yay paru apk
   emerge portage brew port pkg pkg_add pkg_delete pkgin nix nix-env nix-shell guix`,
  // Android and embedded environments
  `adb fastboot am pm content settings device_config dumpsys getprop setprop logcat logd ime input
   monkey app_process appops svc wm dpm toybox busybox`,
  // macOS utilities
  `open pbcopy pbpaste osascript defaults launchctl diskutil softwareupdate sw_vers xcode-select xcrun
   xcodebuild spctl csrutil mdls mdfind`,
  // Windows CMD and system tools
  `cmd powershell pwsh cls copy xcopy robocopy del erase ren move md rd type findstr setx chdir pushd
   popd path prompt ver vol tree attrib cacls icacls takeown assoc ftype driverquery fsutil gpresult
   gpupdate ipconfig net netsh pathping powercfg query reg regsvr32 sc schtasks sfc systeminfo tasklist
   taskkill tracert wmic winget choco scoop dism bcdedit comp compact fc format label subst msiexec
   rundll32 control start choice pause call setlocal endlocal goto color title verify`,
  // Development, build and version-control tools
  `git gitk gh glab hub svn hg cvs node nodejs npm npx corepack pnpm yarn bun deno python python2
   python3 pip pip3 pipenv poetry uv conda mamba virtualenv ruby gem rake rails bundle bundler perl php
   composer go gofmt godoc gcc g++ clang clang++ make cmake ctest cpack ninja meson autoconf automake m4
   pkg-config cargo rustc rustup rustfmt java javac jar javadoc jshell mvn gradle kotlin kotlinc dotnet
   msbuild csc mono swift swiftc code code-insiders cursor`,
  // Databases, containers, infrastructure and cloud CLIs
  `mysql mysqladmin mariadb psql pg_dump pg_restore createdb dropdb sqlite3 redis-cli mongod mongo
   mongosh influx sqlcmd docker docker-compose podman buildah nerdctl kubectl helm minikube kind
   kustomize terraform tofu terragrunt vagrant ansible ansible-playbook packer aws az gcloud gsutil
   azcopy doctl heroku flyctl vercel netlify wrangler supabase firebase serverless sls`,
]

const COMMANDS = new Set(
  COMMAND_GROUPS.flatMap((group) => group.trim().split(/\s+/)).map((command) => command.toLowerCase()),
)

const EXECUTABLE_SUFFIX = /\.(?:exe|com|bat|cmd|ps1|sh)$/i
const ENV_ASSIGNMENT = /^[A-Za-z_][A-Za-z0-9_]*=/
const POWERSHELL_VERBS = new Set(
  `add approve assert backup block checkpoint clear close compare complete confirm convert convertfrom
   convertto copy debug deny disable disconnect dismount enable enter exit expand export find format get
   grant group hide import initialize install invoke join limit lock measure merge mount move new open
   optimize out ping protect publish push read receive redo register remove rename repair request reset
   resize resolve restart restore resume revoke save search select send set show skip split start step
   stop submit suspend switch sync test trace unblock unprotect unregister update use wait watch write`
    .split(/\s+/),
)

interface ScanResult {
  segments: string[]
  hasShellSyntax: boolean
}

function readSubstitution(input: string, start: number) {
  let depth = 1
  let quote = ''
  let index = start + 1
  while (index < input.length) {
    const char = input[index]
    if (char === '\\' && quote !== "'") {
      index += 2
      continue
    }
    if (quote) {
      if (char === quote) quote = ''
    } else if (char === "'" || char === '"' || char === '`') {
      quote = char
    } else if (char === '(' && input[index - 1] === '$') {
      depth += 1
    } else if (char === ')') {
      depth -= 1
      if (!depth) return { content: input.slice(start + 1, index), end: index + 1 }
    }
    index += 1
  }
  return { content: input.slice(start + 1), end: input.length }
}

function readBacktick(input: string, start: number) {
  let index = start + 1
  while (index < input.length) {
    if (input[index] === '\\') {
      index += 2
      continue
    }
    if (input[index] === '`') return { content: input.slice(start + 1, index), end: index + 1 }
    index += 1
  }
  return { content: input.slice(start + 1), end: input.length }
}

function scanSegments(input: string): ScanResult {
  const segments = ['']
  let quote = ''
  let hasShellSyntax = false
  let index = 0

  while (index < input.length) {
    const char = input[index]
    const next = input[index + 1]
    if (char === '\\' && quote !== "'" && next) {
      segments[segments.length - 1] += char + next
      index += 2
      continue
    }
    if (quote) {
      if (char === quote) quote = ''
      segments[segments.length - 1] += char
      index += 1
      continue
    }
    if (char === "'" || char === '"') {
      quote = char
      segments[segments.length - 1] += char
      index += 1
      continue
    }
    if (char === '$' && next === '(') {
      const substitution = readSubstitution(input, index + 1)
      hasShellSyntax = true
      segments[segments.length - 1] += ' '
      index = substitution.end
      continue
    }
    if (char === '`') {
      const substitution = readBacktick(input, index)
      hasShellSyntax = true
      segments[segments.length - 1] += ' '
      index = substitution.end
      continue
    }
    if (char === '\n' || char === '\r' || char === ';' || char === '&' || char === '|' || char === '<' || char === '>') {
      if ((char === '&' && next === '&') || (char === '|' && next === '|') || (char === '>' && next === '>')) index += 1
      hasShellSyntax = true
      segments.push('')
      index += 1
      continue
    }
    segments[segments.length - 1] += char
    index += 1
  }
  return { segments, hasShellSyntax }
}

function tokenize(segment: string) {
  const tokens: string[] = []
  let token = ''
  let quote = ''
  let started = false
  let index = 0
  while (index < segment.length) {
    const char = segment[index]
    const next = segment[index + 1]
    if (char === '\\' && quote !== "'" && next && /[\s'"`$&|;<>\\]/.test(next)) {
      token += next
      started = true
      index += 2
      continue
    }
    if (quote) {
      if (char === quote) quote = ''
      else token += char
      started = true
      index += 1
      continue
    }
    if (char === "'" || char === '"') {
      quote = char
      started = true
      index += 1
      continue
    }
    if (/\s/.test(char)) {
      if (started) tokens.push(token)
      token = ''
      started = false
      index += 1
      continue
    }
    token += char
    started = true
    index += 1
  }
  if (started) tokens.push(token)
  return tokens
}

function commandBasename(token: string) {
  return token.split(/[\\/]/).pop()?.toLowerCase() || ''
}

function isPowerShellCommand(name: string) {
  const match = name.match(/^([a-z]+)-[a-z][a-z0-9-]*$/i)
  return Boolean(match && POWERSHELL_VERBS.has(match[1].toLowerCase()))
}

function isCommandHead(token: string) {
  const name = commandBasename(token)
  const withoutSuffix = name.replace(EXECUTABLE_SUFFIX, '')
  return COMMANDS.has(name)
    || COMMANDS.has(withoutSuffix)
    || EXECUTABLE_SUFFIX.test(name)
    || isPowerShellCommand(withoutSuffix)
    || /^(?:python|pip)(?:\d+(?:\.\d+)*)?$/.test(name)
    || /^(?:gcc|g\+\+|clang|clang\+\+)-\d+$/.test(name)
}

function isPathInvocation(token: string) {
  if (/^[A-Za-z][A-Za-z0-9+.-]*:\/\//.test(token)) return false
  return token.includes('/') || token.includes('\\') || /^[A-Za-z]:/.test(token)
}

function segmentIsCommand(segment: string) {
  const tokens = tokenize(segment)
  let index = 0
  while (index < tokens.length && ENV_ASSIGNMENT.test(tokens[index])) index += 1
  if (!tokens.length) return false
  if (index > 0) return true
  const head = tokens[index]
  return isCommandHead(head) || isPathInvocation(head) || Boolean(tokens[index + 1]?.startsWith('-'))
}

export function isShellCommand(input: string): boolean {
  if (!input.trim()) return false
  const { segments, hasShellSyntax } = scanSegments(input)
  if (hasShellSyntax) return true
  return segments.some(segmentIsCommand)
}

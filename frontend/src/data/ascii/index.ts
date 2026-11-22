import arch from './arch'
import ubuntu from './ubuntu'
import debian from './debian'
import mint from './mint'
import fedora from './fedora'
import manjaro from './manjaro'
import endeavouros from './endeavouros'
import opensuse from './opensuse'
import kali from './kali'
import gentoo from './gentoo'
import alpine from './alpine'
import nixos from './nixos'
import voidArt from './void'
import popos from './popos'
import elementary from './elementary'
import zorin from './zorin'
import rocky from './rocky'
import almalinux from './almalinux'
import centos from './centos'
import rhel from './rhel'
import deepin from './deepin'
import windows from './windows'
import macos from './macos'
import android from './android'
import ios from './ios'
import ipados from './ipados'
import chromeos from './chromeos'
import freebsd from './freebsd'
import linux from './linux'
import unknown from './unknown'

export const asciiArt: Record<string, string> = {
  arch, ubuntu, debian, mint, fedora, manjaro, endeavouros, opensuse,
  kali, gentoo, alpine, nixos, void: voidArt, popos, elementary, zorin,
  rocky, almalinux, centos, rhel, deepin, windows, macos, android, ios,
  ipados, chromeos, freebsd, linux, unknown,
}

export function getAsciiArt(distribution: string): string {
  return asciiArt[distribution] || unknown
}

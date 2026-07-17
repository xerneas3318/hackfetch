# hackfetch-bin (AUR)

Source of truth for the [`hackfetch-bin` AUR package](https://aur.archlinux.org/packages/hackfetch-bin).

Installs the pre-built Linux binary from the GitHub release, so users on Arch
(and CachyOS / Manjaro / EndeavourOS / any Arch derivative) can grab hackfetch
via any AUR helper:

```sh
yay -S hackfetch-bin
paru -S hackfetch-bin
```

## Updating for a new release

When a new hackfetch tag ships, update this package with the new version and
its release binary shasums, then push to the AUR repo. Steps:

1. Bump `pkgver=` in `PKGBUILD` (and reset `pkgrel=1`).
2. Grab the new sha256s:
   ```sh
   V=1.7.3   # replace with the new tag
   curl -fsSL https://github.com/xerneas3318/hackfetch/releases/download/v${V}/hackfetch-linux-amd64.tar.gz | sha256sum
   curl -fsSL https://github.com/xerneas3318/hackfetch/releases/download/v${V}/hackfetch-linux-arm64.tar.gz | sha256sum
   ```
3. Paste the sha256s into `sha256sums_x86_64=` and `sha256sums_aarch64=` in `PKGBUILD`.
4. Regenerate `.SRCINFO`:
   ```sh
   cd packaging/aur/hackfetch-bin
   makepkg --printsrcinfo > .SRCINFO
   ```
5. Commit both files here, then mirror them to the AUR git repo (see below).

## First-time publish to AUR

Only done once. If the package is already on AUR, skip this and go to "Push
updates" below.

1. Make an account at <https://aur.archlinux.org/> and add your SSH public
   key under Account Details.
2. Clone the (empty) AUR repo somewhere outside this repo:
   ```sh
   git clone ssh://aur@aur.archlinux.org/hackfetch-bin.git ~/aur/hackfetch-bin
   ```
3. Copy `PKGBUILD` and `.SRCINFO` from this folder into that clone.
4. Commit and push:
   ```sh
   cd ~/aur/hackfetch-bin
   git add PKGBUILD .SRCINFO
   git commit -m "initial import of hackfetch-bin 1.7.2"
   git push
   ```

## Push updates

Every time `pkgver` bumps:

```sh
cp packaging/aur/hackfetch-bin/{PKGBUILD,.SRCINFO} ~/aur/hackfetch-bin/
cd ~/aur/hackfetch-bin
git add PKGBUILD .SRCINFO
git commit -m "hackfetch-bin ${NEW_VERSION}"
git push
```

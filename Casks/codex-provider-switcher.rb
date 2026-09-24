cask "codex-provider-switcher" do
  version "0.4.1"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.1/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "ebfa5e7a3268888517f9316b145be84f5829f7edd60ad232f992a21aca812848"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.1/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "61b10c9c4d7a134e28123c541fb6d03f6ca05570c104fab9bd7e92aa0c0b8acb"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

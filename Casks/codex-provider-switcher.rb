cask "codex-provider-switcher" do
  version "0.4.3"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.3/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "a26c4aec318c64840aa05b8efc2762cb5c9c82faf0882550f54d424de72b9c13"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.3/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "dd667ce2657798533408223288d740de419a7e35227d622b57ef7242ffa86c1a"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

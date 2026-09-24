cask "codex-provider-switcher" do
  version "0.3.0"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.3.0/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "5aff80f10d79a1f64d5e95a6401cdffe5060796abd096789a5d7ea07399cd316"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.3.0/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "61e5d4d73e4927df8f8d3a26340186e13777d65acb5eb643fc8465bb3df1fb8a"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

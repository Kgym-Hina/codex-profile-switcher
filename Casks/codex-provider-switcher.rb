cask "codex-provider-switcher" do
  version "0.4.0"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.0/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "f86e222d7c447529e8eb078bf2ff3f0e7d8ec63bca5022d0855f51f59162e0e3"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.0/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "3f59149fd612eca51df85c88fa0efefbb300d7d3610f042460172062c79f82ac"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

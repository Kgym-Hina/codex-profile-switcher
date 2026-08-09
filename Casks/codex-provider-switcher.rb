cask "codex-provider-switcher" do
  version "0.2.0"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.2.0/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "211672d69a584e5f92a27ccdc08bcf9ec3fee507d7cfd2ce4c32560c781b45cf"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.2.0/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "47f261aebd39930c27d8eb60501aa99bb634dee94b244c7d11e42090eac5b252"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

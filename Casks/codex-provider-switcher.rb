cask "codex-provider-switcher" do
  version "0.1.0"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.1.0/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "ecec6f1c791f953096a53e24f6f3c131a4c305c1e231da217b913048d42c0fdf"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.1.0/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "d13952842dd4a19706239dc3cc96418fdf5dfddf06e8fddd4dbfcdc10ddffbad"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

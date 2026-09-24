cask "codex-provider-switcher" do
  version "0.4.2"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.2/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "3f521a387314439b1429897042d5be303e09812b631cb52fb2001281d99115f1"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.4.2/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "386e2f76ed4aef95c2c7acefae0bd5dd7458f2a948f5cb0909e76626869c1111"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

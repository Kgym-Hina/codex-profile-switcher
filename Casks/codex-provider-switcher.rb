cask "codex-provider-switcher" do
  version "0.1.1"

  on_arm do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.1.1/codex-provider-switcher-darwin-arm64.tar.gz"
    sha256 "83eabea05f0e1c90a71b60fe0503011aa77a9474448fec646f71569e86528bf9"
  end

  on_intel do
    url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v0.1.1/codex-provider-switcher-darwin-amd64.tar.gz"
    sha256 "b3628894f20b32f1812b0828dd4a0de55e44d866cad906c250643a92b2dd50ca"
  end

  name "Codex Profile Switcher"
  desc "TUI tool for switching Codex provider profiles"
  homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

  binary "codex-provider-switch"
end

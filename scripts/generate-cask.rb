#!/usr/bin/env ruby

version, arm_sha, intel_sha = ARGV
if [version, arm_sha, intel_sha].any? { |value| value.nil? || value.empty? }
  warn "usage: generate-cask.rb VERSION ARM64_SHA256 INTEL_SHA256"
  exit 1
end

unless [arm_sha, intel_sha].all? { |value| value.match?(/\A[0-9a-f]{64}\z/) }
  warn "invalid SHA256"
  exit 1
end

Dir.mkdir("Casks") unless Dir.exist?("Casks")
File.write("Casks/codex-provider-switcher.rb", <<~RUBY)
  cask "codex-provider-switcher" do
    version "#{version}"

    on_arm do
      url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v#{version}/codex-provider-switcher-darwin-arm64.tar.gz"
      sha256 "#{arm_sha}"
    end

    on_intel do
      url "https://github.com/Kgym-Hina/codex-profile-switcher/releases/download/v#{version}/codex-provider-switcher-darwin-amd64.tar.gz"
      sha256 "#{intel_sha}"
    end

    name "Codex Profile Switcher"
    desc "TUI tool for switching Codex provider profiles"
    homepage "https://github.com/Kgym-Hina/codex-profile-switcher"

    binary "codex-provider-switch"
  end
RUBY

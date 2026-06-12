class Hackfetch < Formula
  desc "Hack Club themed system fetch with Hackatime stats"
  homepage "https://github.com/xerneas3318/hackfetch"
  url "https://github.com/xerneas3318/hackfetch/archive/refs/tags/v0.1.4.tar.gz"
  sha256 "54113bdd3207f919da82ab700fb751a38249b46f45b4910ecc4ff4b43ebb5366"
  license "MIT"
  head "https://github.com/xerneas3318/hackfetch.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  test do
    output = shell_output("#{bin}/hackfetch -list")
    assert_match "hackclub", output
  end
end

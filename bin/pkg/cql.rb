class Cql < Formula
  desc "Decentralized SQL database with blockchain features"
  homepage "https://covenantsql.io"
  version "v0.5.0"
  url "https://github.com/CovenantSQL/CovenantSQL/archive/#{version}.tar.gz"
  sha256 "ebee3e8fca672d6f3d3d607e380ec7a1f3a31501e5c80b85c660f50eda7a2240"
  head "https://github.com/CovenantSQL/CovenantSQL.git"

  depends_on "go" => :build
  depends_on "make" => :build

  def install
    ENV["GOPATH"] = buildpath
    ENV["CQLVERSION"] = version
    mkdir_p "src/github.com/CovenantSQL"
    ln_s buildpath, "src/github.com/CovenantSQL/CovenantSQL"
    system "make", "bin/cql"
    bin.install "bin/cql"
  end

  test do
    system bin/"cql", "help"
  end
end
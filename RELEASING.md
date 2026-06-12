# Releasing

The source repository is `github.com/feint/cli-todo`. The Homebrew tap repository is `github.com/feint/homebrew-todo`.

## Homebrew Packaging

1. Tag the source release:

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. Create a GitHub release for the tag.

3. Calculate the source archive checksum:

   ```sh
   curl -L https://github.com/feint/cli-todo/archive/refs/tags/v0.1.0.tar.gz | shasum -a 256
   ```

4. Add or update `Formula/cli-todo.rb` in `github.com/feint/homebrew-todo`:

   ```ruby
   class CliTodo < Formula
     desc "Tiny project-local todo CLI"
     homepage "https://github.com/feint/cli-todo"
     url "https://github.com/feint/cli-todo/archive/refs/tags/v0.1.0.tar.gz"
     sha256 "PASTE_SHA256_HERE"
     license "MIT"

     depends_on "go" => :build

     def install
       ldflags = "-s -w -X main.version=#{version}"
       system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/cli-todo"
       bin.install_symlink "cli-todo" => "todo"
     end

     test do
       assert_match "cli-todo #{version}", shell_output("#{bin}/cli-todo --version")
       assert_match "cli-todo #{version}", shell_output("#{bin}/todo --version")
     end
   end
   ```

5. Test the formula:

   ```sh
   brew install --build-from-source feint/todo/cli-todo
   brew test feint/todo/cli-todo
   ```

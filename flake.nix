{
  description = "my project description";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let pkgs = nixpkgs.legacyPackages.${system}; in
        {
          devShells.default = with pkgs; mkShell {
            buildInputs = [gopls delve go];
            hardeningDisable = [ "all" ];
            shellHook = ''
              echo Welcome to langekko devshell!
              echo To build and run the project:
              echo "rm language.db || true && go run cmd/main.go"
            '';
            };
        }
      );
}

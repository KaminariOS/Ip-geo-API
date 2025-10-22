{
  description = "A Nix-flake-based Go development environment with pre-commit shell hook";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    git-hooks-nix.inputs.nixpkgs.follows = "nixpkgs";
    git-hooks-nix.url = "github:cachix/git-hooks.nix";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {
    self,
    flake-parts,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      # bring in git-hooks.nix's flake module for easy pre-commit setup
      imports = [inputs.git-hooks-nix.flakeModule];

      perSystem = {
        config,
        pkgs,
        lib,
        system,
        ...
      }: let
        # Pick a Go toolchain (falls back gracefully if attr name changes)
        goPkg = pkgs.go;

        # Common Go dev tools
        goTools = with pkgs; [
          goPkg
          gopls
          gotools
          golangci-lint
          delve
        ];
      in {
        # Configure pre-commit via git-hooks.nix
        pre-commit.settings = {
          # Keep universally-helpful text hooks
          hooks = {
            end-of-file-fixer.enable = true;
            # Custom Go hooks via "system" language
            go-fmt = {
              enable = true;
              name = "go fmt";
              entry = "${goPkg}/bin/gofmt -l -w";
              language = "system";
              files = "\\.go$";
            };
            golangci-lint = {
              # enable = true;
              name = "golangci-lint";
              entry = "${pkgs.golangci-lint}/bin/golangci-lint run --fix=false";
              language = "system";
              files = "\\.go$";
            };
          };

          # Make sure hooks have the right tools available
          enabledPackages = goTools;
        };

        devShells.default = pkgs.mkShell {
          name = "go-dev";

          packages =
            goTools
            ++ config.pre-commit.settings.enabledPackages;

          # Install the pre-commit hook on shell entry
          shellHook = config.pre-commit.installationScript;
        };
      };

      flake = {};
    };
}

{
  description = "orgo - browse your org-roam notes in the browser, with a link graph";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "orgo";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-JqOv6bcTKfwmK34eJeXhJHs+6o71uTkB0W834MfCFVA=";
          ldflags = [
            "-s"
            "-w"
            "-X main.version=0.1.0"
          ];
          meta.mainProgram = "orgo";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools # staticcheck
            nodejs_22
            nixfmt-rfc-style
          ];
        };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}

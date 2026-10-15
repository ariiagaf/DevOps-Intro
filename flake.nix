{
  description = "Reproducible QuickNotes builds";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";

  outputs = { self, nixpkgs }:
    let
      system = "aarch64-darwin";
      pkgs = import nixpkgs { inherit system; };

      quicknotes = pkgs.buildGoModule {
        pname = "quicknotes";
        version = "1.0.0";
        src = ./app;

        vendorHash = null;
        env.CGO_ENABLED = "0";
        ldflags = [ "-s" "-w" ];
      };
    in
    {
      packages.${system} = {
        quicknotes = quicknotes;
        default = quicknotes;
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [
          go
          gopls
          golangci-lint
        ];
      };
    };
}

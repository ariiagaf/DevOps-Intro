{
  description = "Reproducible QuickNotes builds";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-linux" ];

      forAllSystems = f:
        nixpkgs.lib.genAttrs systems
          (system: f (import nixpkgs { inherit system; }));

      mkQuicknotes = pkgs: pkgs.buildGoModule {
        pname = "quicknotes";
        version = "1.0.0";
        src = ./app;

        vendorHash = null;
        env.CGO_ENABLED = "0";
        ldflags = [ "-s" "-w" ];
      };
    in
    {
      packages = forAllSystems (pkgs:
        let
          quicknotes = mkQuicknotes pkgs;
        in
        {
          inherit quicknotes;
          default = quicknotes;

          docker = pkgs.dockerTools.buildImage {
            name = "quicknotes";
            tag = "nix";

            copyToRoot = pkgs.buildEnv {
              name = "quicknotes-root";
              paths = [ quicknotes pkgs.cacert ];
              pathsToLink = [ "/bin" "/etc" ];
            };

            extraCommands = ''
              mkdir -p data
              chmod 777 data
            '';

            config = {
              WorkingDir = "/data";
              Entrypoint = [ "${quicknotes}/bin/quicknotes" ];
              ExposedPorts = {
                "8080/tcp" = {};
              };
              User = "10001:10001";
              Env = [
                "PORT=8080"
                "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt"
              ];
            };
          };
        });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            golangci-lint
          ];
        };
      });
    };
}
{
  description = "real-time clipboard monitoring for sensitive information";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-linux" "aarch64-linux" ];
      forEachSystem = nixpkgs.lib.genAttrs systems;

      clipleaksFor = pkgs: pkgs.callPackage ./package.nix { src = self; };
    in
    {
      packages = forEachSystem (system:
        let clipleaks = clipleaksFor nixpkgs.legacyPackages.${system};
        in { inherit clipleaks; default = clipleaks; });

      overlays.default = final: _prev: { clipleaks = clipleaksFor final; };
    };
}

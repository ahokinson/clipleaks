{ lib, buildGoModule, src }:

let
  version = "0.1.2";
in
buildGoModule {
  pname = "clipleaks";

  inherit version src;

  vendorHash = null;

  subPackages = [ "cmd/clipleaks" ];

  # main.version/main.commit default to "dev"/"none"; without this the
  # binary reports those instead of what was built.
  ldflags = [ "-X main.version=${version}" ];

  meta = {
    description = "real-time clipboard monitoring for sensitive information (API keys, tokens, private keys)";
    homepage = "https://github.com/ahokinson/clipleaks";
    license = lib.licenses.mit;
    mainProgram = "clipleaks";
  };
}

{ system ? builtins.currentSystem, pkgs ? import <nixpkgs> { inherit system; } }:
let
  inherit (pkgs) lib;

in pkgs.mkShell {
  name = "refreshment";
  buildInputs = [
    pkgs.go
  ];
}


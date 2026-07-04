#!/usr/bin/env bash
# Hook de muestra del lab: deja un rastro para comprobar que se ejecutó.
echo "hook post-link del lab, perfil=$DOTS_PROFILE" >"$DOTS_TARGET/.lab-hook"

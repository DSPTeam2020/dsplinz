#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
dspdir="$workspace/src/github.com/dsplinz2019"
if [ ! -L "$dspdir/dsp" ]; then
    mkdir -p "$dspdir"
    cd "$dspdir"
    ln -s ../../../../../. dsp
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$dspdir/dsp"
PWD="$dspdir/dsp"

# Launch the arguments with the configured environment.
exec "$@"

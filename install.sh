#!/bin/bash

set -e

add_to_path() {
  shell="$SHELL";
  rcfile=".bashrc"
  if [[ "$shell" == *"zsh" ]]; then
    rcfile=".zshrc";
  fi
  if grep -q "/.altlimit/bin" "$HOME/$rcfile"; then
    echo "~/.altlimit/bin already in zsh path"
  else
    echo "Adding ~/.altlimit/bin to PATH in $rcfile";
    echo 'export PATH=$PATH:$HOME/.altlimit/bin' >> $HOME/$rcfile;
    echo "Restart your terminal or run 'source ~/$rcfile'"
  fi
}

install_binary() {
  if [[ ! -d $HOME/.altlimit/bin ]]; then
    echo "Making $HOME/.altlimit/bin"
    mkdir -p $HOME/.altlimit/bin;
  fi
  echo "Downloading latest statusalert binary at: $1";
  curl -o $HOME/.altlimit/bin/statusalert.tgz -s -S -L "$1"
  tar -xf $HOME/.altlimit/bin/statusalert.tgz -C $HOME/.altlimit/bin/
  rm $HOME/.altlimit/bin/statusalert.tgz
  if [ $? -ne 0 ]; then
    echo "Download failed.";
    exit 1;
  fi
  chmod +x $HOME/.altlimit/bin/statusalert;
  add_to_path;
  echo "statusalert has been installed at $HOME/.altlimit/bin";
}

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
       MACHINE_TYPE=`uname -m`
       if [ ${MACHINE_TYPE} == 'x86_64' ]; then
          install_binary "https://github.com/altlimit/statusalert/releases/download/latest/linux.tgz"
       else
          echo "${MACHINE_TYPE} not supported. Try building from source.";
          exit 1;
       fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
        # Mac OSX
       MACHINE_TYPE=`uname -m`
       if [ ${MACHINE_TYPE} == 'x86_64' ]; then
          install_binary "https://github.com/altlimit/statusalert/releases/download/latest/darwin.tgz"
       else
          echo "${MACHINE_TYPE} not supported. Try building from source.";
          exit 1;
       fi
else
       echo "$OSTYPE not yet supported.";
       exit 1;
fi

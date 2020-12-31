
Build [SRT](https://github.com/Haivision/srt)

    git clone https://github.com/Haivision/srt.git
    sudo apt install tclsh pkg-config cmake libssl-dev build-essential
    ./configure
    make
    sudo make install

Build [srt live server](https://github.com/Edward-Wu/srt-live-server)

    git clone https://github.com/Edward-Wu/srt-live-server.git
    sudo apt install zlib1g-dev
    make

Build [FFmpeg](https://github.com/FFmpeg/FFmpeg)

Note:
1. Server users can omit the `ffplay` and x11grab dependencies: `libsdl2-dev`,
`libva-dev`, `libvdpau-dev`, `libxcb1-dev`, `libxcb-shm0-dev`, `libxcb-xfixes0-dev`.
2. Must use snapshot of 4.3

Build Steps

    git checkout -b local origin/release/4.3
    sudo apt install autoconf automake build-essential cmake git-core \
      libass-dev libfreetype6-dev libgnutls28-dev libsdl2-dev libtool \
      libva-dev libvdpau-dev libvorbis-dev libxcb1-dev libxcb-shm0-dev \
      libxcb-xfixes0-dev pkg-config texinfo wget yasm zlib1g-dev \
      nasm libx264-dev libx265-dev libnuma-dev libvpx-dev libfdk-aac-dev \
      libmp3lame-dev libopus-dev
    ./configure --enable-srt
    make
    sudo make install

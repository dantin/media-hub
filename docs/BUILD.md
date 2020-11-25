
Build [SRT](https://github.com/Haivision/srt)

    git clone https://github.com/Haivision/srt.git
    sudo apt update
    sudo apt upgrade
    sudo apt install tclsh pkg-config cmake libssl-dev build-essential
    ./configure
    make
    sudo make install

Build [srt live server](https://github.com/Edward-Wu/srt-live-server)

    git clone https://github.com/Edward-Wu/srt-live-server.git
    sudo apt install zlib1g-dev
    make

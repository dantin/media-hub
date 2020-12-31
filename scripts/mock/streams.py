# -*- coding: utf-8 -*-                                                                                                
 
import argparse
import logging
import os
import signal
import subprocess
import sys
import time

from threading import Event, Thread


LOGGER = logging.getLogger('ROOT')

# shutdown event
_shutdown_event = Event()


def parse_args():
    """parse_args parses command line arguments."""
    parser = argparse.ArgumentParser('streams.py')
    parser.add_argument('config', help='configuration file')
    parser.add_argument('-L', '--level', choices=('debug', 'info', 'warn'), default='info', help='log level: debug, info, warn')
    parser.add_argument('-V', '--version', help='print version information', action='store_true')

    args = parser.parse_args()

    if args.version:
        print('mock video streams, version', '0.1-dev')
        sys.exit(0)
           
    # setup logging.
    default_handler = logging.StreamHandler()
    default_handler.setFormatter(logging.Formatter(
        '[%(asctime)s] %(levelname)s: %(message)s'
        ))
    LOGGER.addHandler(default_handler)
    if args.level == 'info':
        LOGGER.setLevel(logging.INFO)
    elif args.level == 'warn':
        LOGGER.setLevel(logging.WARN)
    else:
        LOGGER.setLevel(logging.DEBUG)

    return load_config(args.config)


def load_config(fname, supported_types=('dev', 'cam')):
    from configparser import ConfigParser

    rooms = []

    if not os.path.exists(fname):
        return rooms

    # read configuration.
    config = ConfigParser()
    config.read(fname)

    host = config.get('global', 'host')
    port = config.getint('global', 'port')
    stream_prefix = config.get('global', 'stream_prefix')
    room_names = config.get('global', 'rooms')

    for name in room_names.split(','):
        room = Room(name)
        for st in supported_types:
            src = config.get(name, st)
            if not os.path.exists(src):
                LOGGER.warning('File "%s" is not exits', src)
                continue
            dest = 'srt://{}:{}?streamid={}/{}_{}'.format(host, port, stream_prefix, name, st)
            room.add_stream(Stream(src, dest))
        rooms.append(room)

    return rooms


class Room(Thread):
    """Room mock a clinic room."""
    def __init__(self, name):
        Thread.__init__(self)

        self.name = name
        self.streams = []

    def run(self):
        LOGGER.info('Room %s sending %d video stream(s)', self.name, len(self.streams))
        for s in self.streams:
            s.start()

    def close(self):
        LOGGER.info('Room %s closing %d video stream(s)', self.name, len(self.streams))
        for s in self.streams:
            s.close()

    def add_stream(self, stream):
        self.streams.append(stream)


class Stream(Thread):
    """Stream do video streaming using `ffmpeg`."""
    def __init__(self, src, dest):
        Thread.__init__(self)
        self.src = src
        self.dest = dest
        self.proc = None

    def run(self):
        if not self.src:
            return

        LOGGER.debug('Streaming file "%s" to SRT server as "%s"', self.src, self.dest)
        cmd = 'ffmpeg -stream_loop -1 -re -i {src} -codec copy -f mpegts {dest} 1>/dev/null 2>&1'.format(src=self.src, dest=self.dest)
        self.proc = subprocess.Popen('exec {}'.format(cmd), shell=True)

    def close(self):
        if self.proc is not None:
            self.proc.kill()


def shutdown_cb(sig, frame):
    """Callback function for shutdown event."""
    LOGGER.info('Recieve signal #%d, shutdown...', sig)
    _shutdown_event.set()


def main():
    """main function."""
    # Register shutdown handler.
    for sig in (signal.SIGINT, signal.SIGTERM):
        signal.signal(sig, shutdown_cb)

    workers = parse_args()

    if not workers:
        LOGGER.warning('No rooms found')
        return

    for w in workers:
        w.start()

    while not _shutdown_event.is_set():
        time.sleep(1.0)

    for w in workers:
        w.close()


if __name__ == '__main__':
    main()
    LOGGER.info('Done!')

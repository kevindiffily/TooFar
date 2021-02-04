# since smbus requires root access
# this small process reads from the smbus and writes to a FIFO for toofar to report on
# I should implement this in C or Go, but this is fine for now
import os, smbus, time, errno

def fetch():
	t_reg = 0x05
	address = 0x18
	bus = smbus.SMBus(1) # change to 0 for older RPi revision
	reading = bus.read_i2c_block_data(address, t_reg)
	t = (reading[0] << 8) + reading[1]

	# calculate temperature (see 5.1.3.1 in datasheet)
	temp = t & 0x0FFF
	temp /=  16.0
	if (t & 0x1000):
    		temp -= 256
	return temp

def main():
	FIFO = "/tmp/tempfifo"
	if not os.path.exists(FIFO):
		print('making %s' % FIFO)
		os.mkfifo(FIFO)

	fifo = os.open(FIFO, os.O_WRONLY)
	while True:
		try:
			temp = fetch()
			os.write(fifo, '%f\n' % temp)
			# TooFar reads as quickly as we update
			time.sleep(60)
		except KeyboardInterrupt:
			print('shutting down\n')
			os.close(fifo)
			os.unlink(FIFO)
			break
		except OSError as oe:
			if oe.errno == errno.EPIPE:
				# print("reader diconnected, resetting %s" % FIFO)
				fifo = os.open(FIFO, os.O_WRONLY)
				temp = fetch()
				os.write(fifo, '%f\n' % temp)
				continue

if __name__ == "__main__":
	main()

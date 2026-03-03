# rtldavis

## rtldavis

### About this Repository

This repository is a fork of [https://github.com/lheijst/rtldavis](https://github.com/lheijst/rtldavis) for use with custom receiver modules. It has been modified in this way.

1) Check for duplicate messages that come in less than 2 seconds after the first


## Installation
This installation is for a Raspberry Pi 3B running Trixie OS, WeeWX 5.2 installed with apt along with a BME280 temperature humidity sensor

There is another installation by Vince Skahan in [https://github.com/weewx-contrib/weewx-rtldavis/blob/main/install-weewx-rtldavis.sh](https://github.com/weewx-contrib/weewx-rtldavis/blob/main/install-weewx-rtldavis.sh)

#### Install packages needed for BME280

as per [https://gitlab.com/mfraz74/bme280wx/](as per https://gitlab.com/mfraz74/bme280wx/)

Enable I2C: Run sudo raspi-config and navigate to Interfacing Options > I2C > Enable.

install RPi.bme280 smbus2

sudo apt update

sudo apt install python3-bme280 python3-smbus2

This requires a weewx program installed

weectl extension install [https://gitlab.com/mfraz74/bme280wx/-/archive/master/bme280wx-master.zip](https://gitlab.com/mfraz74/bme280wx/-/archive/master/bme280wx-master.zip)

Weectl should come back with   
1. Saving installer file to /etc/weewx/bin/user/installer/bme280wx

2. Saved copy of configuration as /etc/weewx/weewx.conf.20260225141412
     
   
#### Install packages needed for rtldavis

sudo apt install golang git cmake librtlsdr-dev

#### Setup Udev Rules
find the vendor id and product id for your SDR dongle

    lsusb

I got

Bus 001 Device 006: ID 0bda:2838 Realtek Semiconductor Corp. RTL2838 DVB-T

The important parts are "0bda" (the vendor id) and "2838" (the product id)


Create a new file as root named /etc/udev/rules.d/20.rtlsdr.rules that contains the following line:

    sudo nano /etc/udev/rules.d/20.rtlsdr.rules
    SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", GROUP="adm", MODE="0666", SYMLINK+="rtl_sdr"

With the vendor and product ids for your particular dongle. This should make the dongle accessible to any user in the adm group. and add a /dev/rtl_sdr symlink when the dongle is attached.

#### Get librtlsdr

    cd /home/pi
    git clone https://github.com/steve-m/librtlsdr.git
    cd librtlsdr
    mkdir build
    cd build
    cmake ../ -DINSTALL_UDEV_RULES=ON -DDETACH_KERNEL_DRIVER=ON
    make
    sudo make install
    sudo ldconfig


#### Change / create ~/.profile file


    sudo nano ~/.profile
    add at the end of the file:
    
    export GOROOT=/usr/lib/go
    export GOPATH=$HOME/work
    export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
    
    
#### Activate changed / new ~/profile
    source ~/.profile

#### Get the rtldavis go package

    cd /home/pi
    git clone https://github.com/LloydR/rtldavis.git
    
This should give you a directory /home/pi/rtldavis    




    
#### Compiling GO sources
     cd /home/pi/rtldavis
     rm -rf vendor/
     go mod tidy
     go install -v .

This should give you a rtldavis go program in /home/pi/work


#### Check that rtldavis go program actually works

     ~/work/bin rtldavis -v 
gave me 0.15 + other information
Now make sure it actually works

    ~/work/bin/rtldavis -v -tr 1 -tf US

should actually start getting data packets from Davis ISS assume your id is 0

#### Modfy weewx.conf

add the following to weewx.conf   
I added it right after Station Information and before the Simulator

weewx.conf is at /etc/weewx

   ```
   [Rtldavis]
       # This section is for the rtldavis sdr-rtl USB receiver.
    
       cmd = /usr/local/bin/rtldavis
       # set the library path to the local build librtlsdr
       ld_library_path = /usr/local/lib
       # Options:
       # -ppm = frequency correction of rtl dongle in ppm; default = 0
       # -gain = tuner gain in tenths of Db; default = 0 means "auto gain"
       # -ex = extra loopTime in ms; default = 0
       # -fc = frequency correction for all channels; default = 0
       # -u  = log undefined signals
       #
       # The options below will autoamically be set
       # -tf = transmitter frequencies, US, NZ or EU
       # -tr = transmitters: tr1=1,  tr2=2,  tr3=4,  tr4=8, 
       #                     tr5=16, tr6=32, tr7=64, tr8=128
    
       # Radio frequency to use between USB transceiver and console: US, NZ or EU
       # US uses 915 MHz, NZ uses 921 MHz and EU uses 868.3 MHz.  Default is EU.
       transceiver_frequency = US
    
       # Used channels: 0=not present, 1-8)
       # The channel of the Vantage Vue ISS or Vantage Pro or Pro2 ISS
       iss_channel = 1
       # The values below only apply for Vantage Pro or Pro2
       anemometer_channel = 0
       leaf_soil_channel = 0
       temp_hum_1_channel = 0
       temp_hum_2_channel = 0
       # rain bucket type (0: 0.01 inch, 1: 0.2 mm)
       rain_bucket_type = 0
    
       # Print debug messages
       # 0=no logging; 1=minimum logging; 2=normal logging; 3=detailed logging
       debug_parse = 0
       debug_rain = 0
       debug_rtld = 3    # rtldavis logging: 1=inf; 2=(1)+data+chan; 3=(2)+pkt
    
       # The pct_good per transmitter can be saved to the database
       # This has only effect with 2 transmitters or more
       save_pct_good_per_transmitter = False
    
       # The driver to use:
       driver = user.rtldavis

   ```
Move the go binary/executable to where weewx expects to see it

     sudo mv /home/pi/work/bin/rtldavis /usr/local/bin/rtldavis 
     sudo chown weewx:weewx /usr/local/bin/rtldavis

Have to change groups WeeWX is subscribed to

     sudo usermod -a -G plugdev,dialout weewx 

Now one has to add rtldavis.py  also see instructions at weewx-rtldavis

### Add rtldavis.py to weewx

    weectl extension install https://github.com/LloydR/weewx-rtldavis/archive/master.zip

In weewx.conf you need to change station_type from Simulator to Rtldavis in the section named [Station]

```
    
    # Set to type of station hardware. There must be a corresponding stanza
    # in this file, which includes a value for the 'driver' option.
    station_type = Rtldavis

```
### Troubleshooting crc error

I ran into a CRC16 error 

        1. unsupported operand type(s) for ^: 'int' and 'str'
        2. File "/usr/share/weewx/weewxd.py", line 127
        3. File "/etc/weewx/bin/user/rtldavis.py", line 995, 616, 628, 525, 609
        4. "/usr/share/weewx/weewx/crc16.py", line 49 

This is how I "fixed", it don't know how many of the "fixes" were necessary

In file /etc/weewx/bin/user/rtldavis.py I changed the 

    def _check_crc(msg):
        if crc16(msg.encode('latin-1') if isinstance(msg, str) else msg) != 0:
            raise ValueError("CRC error")

That didn't fix the problem so I had to change /usr/share/weewx/weewx/crc16.py

```
def crc16(byte_buf, crc_start=0):
#    """ Calculate CRC16 sum"""

#    return reduce(lambda crc_sum, ch: (_table[(crc_sum >> 8) ^ ch] ^ (crc_sum << 8)) & 0xffff,
#                  byte_buf, crc_start)
#AI told me to replace the above with this line for the crc error, the line below didn't work so gave me the below that
#    return reduce(lambda crc_sum, ch: (_table[(crc_sum >> 8) ^ (ch if isinstance(ch, int) else ord(ch))] ^ (crc_sum << 8)) & 0xffff, data, 0)
def crc16(data):
    """ Calculate CRC16 sum"""
    # This specifically fixes the Python 3.13 + rtldavis TypeError
    if isinstance(data, (str, bytes, bytearray)):
        # Iterating over bytes/bytearray in Py3 returns ints
        # Iterating over str returns chars, so we use ord() as a safety net
        return reduce(lambda crc_sum, ch: (_table[(crc_sum >> 8) ^ (ch if isinstance(ch, int) else ord(ch))] ^ (crc_sum << 8)) & 0xffff, data, 0)
    return 0


```




## END

## ORIGINAL INSTRUCTIONS

This repository is a fork of [https://github.com/bemasher/rtldavis](https://github.com/bemasher/rtldavis) for use with custom receiver modules. It has been modified in numerous ways.
1) Added EU frequencies
2) Handling of more than one concurrent transmitters.
3) Output format is changed for use with the weewx-rtldavis driver which does the data parsing. 


## ORIGINAL Installation  INSTRUCTIONS

#### Install packages needed for rtldavis

    sudo apt-get install golang git cmake librtlsdr-dev

#### Setup Udev Rules

Next, you need to add some udev rules to make the dongle available for the non-root users. First you want to find the vendor id and product id for your dongle.
The way I did this was to run:

    lsusb

The last line was the Realtek dongle:
    Bus 001 Device 008: ID 0bda:2838 Realtek Semiconductor Corp.
    Bus 001 Device 005: ID 0bda:2838 Realtek Semiconductor Corp. RTL2838 DVB-T

The important parts are "0bda" (the vendor id) and "2838" (the product id).

Create a new file as root named /etc/udev/rules.d/20.rtlsdr.rules that contains the following line:
    nano /etc/udev/rules.d/20.rtlsdr.rules
    SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", GROUP="adm", MODE="0666", SYMLINK+="rtl_sdr"

With the vendor and product ids for your particular dongle. This should make the dongle accessible to any user in the adm group. and add a /dev/rtl_sdr symlink when the dongle is attached.

#### Get librtlsdr

    cd /home/pi
    git clone https://github.com/steve-m/librtlsdr.git
    cd librtlsdr
    mkdir build
    cd build
    cmake ../ -DINSTALL_UDEV_RULES=ON -DDETACH_KERNEL_DRIVER=ON
    make
    sudo make install
    sudo ldconfig

#### Change / create ~/profile

    sudo nano ~/.profile
    add at the end of the file:
    
    export GOROOT=/usr/lib/go
    export GOPATH=$HOME/work
    export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
    
#### Activate changed / new ~/profile
    source ~/.profile

#### Get the rtldavis package

    cd /home/pi
    go get -v github.com/lheijst/rtldavis

#### Compiling GO sources

    cd $GOPATH/src/github.com/lheijst/rtldavis
    git submodule init
    git submodule update
    go install -v .

#### Start program rtldavis

    $GOPATH/bin/rtldavis

#### Usage

Available command-line flags are as follows:

```
Usage of rtldavis:

  -tr [transmitters]
    	code of the stations to listen for: 
        tr1=1 tr2=2 tr3=4 tr4=8 tr5=16 tr6=32 tr7=64 tr8=128
        or the Davis syntax (first transmitter ID has value 0):
        ID 0=1 ID 1=2 ID 2=4 ID 3=8 ID 4=16 ID 5=32 ID 6=64 ID 7=128
        When two or more transmitters are combined, add the numbers.
        Example: ID0 and ID2 combined is 1 + 4 => -tr 5
        
        Default = -tr 1 (ID 0)

  -tf [tranceiver frequencies]
        EU or US
        Default = -tf EU

  -ex [extra loop_delay in ms]
        In case a lot of messages are missed we might try to use the -ex parameter, like -ex 200
        Note: A negative value will probably lead to message loss
        Default = -ex 0
 
  -fc [frequency correction in Hz for all channels]
        Default = -fc 0
        
  -ppm [frequency correction of rtl dongle in ppm]
        Default = -ppm 0
        
  -maxmissed [max missed-packets-in-a-row before new init]
        Normally you should set this parameter to 4 (-maxmissed 4). 
        During testing of new hardware it may be handy (for US equipment) to leave the default value of 51. 
        The program hops along all channels and present information about each individual channel. 
        Default = -maxmissed 51
        
  -u [log undefined signals]
        The program can pick up (i.e. reveive) messages from undefined transmitters, e.g. from a weather 
        station near-by. De messages are discarded, but you may want to see on which channels they are 
        received and how many.
        Default = -u false
```

### License

The source of this project is licensed under GPL v3.0. According to [http://choosealicense.com/licenses/gpl-3.0/](http://choosealicense.com/licenses/gpl-3.0/) you may:

#### Required:

 * **Disclose Source:** Source code must be made available when distributing the software. In the case of LGPL and 
 OSL 3.0, the source for the library (and not the entire program) must be made available.
 * **License and copyright notice:** Include a copy of the license and copyright notice with the code.
 * **State Changes:** Indicate significant changes made to the code.

#### Permitted:

 * **Commercial Use:** This software and derivatives may be used for commercial purposes.
 * **Distribution:** You may distribute this software.
 * **Modification:** This software may be modified.
 * **Patent Use:** This license provides an express grant of patent rights from the contributor to the recipient.
 * **Private Use:** You may use and modify the software without distributing it.

#### Forbidden:

 * **Hold Liable:** Software is provided without warranty and the software author\/license owner cannot be held liable for damages.

### Feedback
If you have any general questions or feedback leave a comment below. For bugs, feature suggestions and anything 
directly relating to the program itself, submit an issue.

# j2n

## Description

Generates C++/JNI wrapper from Java archive file (.jar).

## Installation

    go get github.com/mtojo/j2n

## Usage

    j2n -i file

Options:

    -i <file>      input jar file
    -o <dir>       output directory; same as input if not specify
    -f             overwrite output directory if already exists
    -x <ext>       header file extension (default: .hpp)
    -c <ext>       source file extension (default: .cpp)
    -p <prefix>    include guard prefix
    -s <suffix>    include guard suffix
    -n <namespace> namespace prefix
    -t <file>      header template file
    -u <file>      source template file
    -l <string>    .clang-format file

## Example

Generates from Android SDK:

    $ j2n -i $ANDROID_SDK_ROOT/platforms/android-21/android.jar -o android-sdk

using NAudio.Wave;

namespace Midi8BitSynthesiser.Tests.TestData;

public static class WaveFileAssertions
{
    public static WaveFileData ReadWaveFile(string path)
    {
        using var reader = new WaveFileReader(path);
        var bytes = new byte[(int)reader.Length];
        var offset = 0;
        while (offset < bytes.Length)
        {
            var read = reader.Read(bytes, offset, bytes.Length - offset);
            if (read == 0)
            {
                break;
            }

            offset += read;
        }

        var samples = new short[bytes.Length / sizeof(short)];
        Buffer.BlockCopy(bytes, 0, samples, 0, bytes.Length);

        return new WaveFileData(
            reader.WaveFormat.SampleRate,
            reader.WaveFormat.Channels,
            reader.WaveFormat.BitsPerSample,
            samples);
    }

    public readonly record struct WaveFileData(int SampleRate, int Channels, int BitsPerSample, short[] Samples);
}

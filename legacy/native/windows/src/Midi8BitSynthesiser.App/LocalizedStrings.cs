using System.Globalization;
using Microsoft.Windows.ApplicationModel.Resources;

namespace Midi8BitSynthesiser.App;

internal static class LocalizedStrings
{
    private static readonly Lazy<ResourceLoader?> Loader = new(CreateLoader);

    public static string Get(string key, string fallback)
    {
        var localized = TryGetString(key);
        return string.IsNullOrWhiteSpace(localized) ? fallback : localized;
    }

    public static string Format(string key, string fallbackFormat, params object?[] args)
    {
        var format = Get(key, fallbackFormat);
        return string.Format(CultureInfo.CurrentCulture, format, args);
    }

    private static string? TryGetString(string key)
    {
        try
        {
            return Loader.Value?.GetString(key);
        }
        catch
        {
            return null;
        }
    }

    private static ResourceLoader? CreateLoader()
    {
        try
        {
            return new ResourceLoader();
        }
        catch
        {
            return null;
        }
    }
}

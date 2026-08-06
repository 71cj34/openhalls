import pyfiglet

def render_spaced(text, space=2):
    f = pyfiglet.Figlet(font='standard')
    rendered = f.renderText(text)

    print(f)

print(render_spaced("", space=3))